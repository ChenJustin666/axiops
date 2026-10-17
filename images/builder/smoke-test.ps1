param(
  [string]$Image = 'axiops/builder:handoff-fix',
  [switch]$Jar,
  [string]$MavenImage = 'maven:3.9-eclipse-temurin-17'
)
$ErrorActionPreference = 'Stop'
$suffix = [Guid]::NewGuid().ToString('N').Substring(0, 10)
$network = "axiops-builder-test-$suffix"
$registry = "axiops-registry-$suffix"
$builder = "axiops-builder-$suffix"
$compiler = "axiops-compiler-$suffix"
$fixture = Join-Path ([IO.Path]::GetTempPath()) "axiops-builder-$suffix"
$utf8 = New-Object System.Text.UTF8Encoding($false)
New-Item -ItemType Directory -Path $fixture | Out-Null
[IO.File]::WriteAllText((Join-Path $fixture 'Dockerfile'), "FROM scratch"+[Environment]::NewLine+"COPY marker /marker"+[Environment]::NewLine, $utf8)
[IO.File]::WriteAllText((Join-Path $fixture 'marker'), 'axiops builder smoke test', $utf8)
if ($Jar) {
  New-Item -ItemType Directory -Path (Join-Path $fixture 'src/main/java') -Force | Out-Null
  $pom = @'
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.axiops.smoke</groupId><artifactId>ad-gateway</artifactId><version>1.0.0</version>
  <properties><maven.compiler.release>17</maven.compiler.release><project.build.sourceEncoding>UTF-8</project.build.sourceEncoding></properties>
  <build><plugins>
    <plugin><groupId>org.apache.maven.plugins</groupId><artifactId>maven-compiler-plugin</artifactId><version>3.13.0</version></plugin>
    <plugin><groupId>org.apache.maven.plugins</groupId><artifactId>maven-jar-plugin</artifactId><version>3.4.2</version>
      <configuration><archive><manifest><mainClass>Main</mainClass></manifest></archive></configuration>
    </plugin>
  </plugins></build>
</project>
'@
  [IO.File]::WriteAllText((Join-Path $fixture 'pom.xml'), $pom, $utf8)
  [IO.File]::WriteAllText((Join-Path $fixture 'src/main/java/Main.java'), 'public class Main { public static void main(String[] args) { System.out.println("AXIOPS_JAR_SMOKE_OK"); } }', $utf8)
  $dockerfile = @'
FROM eclipse-temurin:17-jre
WORKDIR /data/apps/ad-gateway
COPY target/ad-gateway-1.0.0.jar app.jar
RUN mkdir -p logs
ENV JAVA_MODULE_OPTS=""
EXPOSE 9099
ENTRYPOINT ["sh", "-c", "java $JAVA_MODULE_OPTS $JAVA_OPTS -jar app.jar"]
'@
  [IO.File]::WriteAllText((Join-Path $fixture 'Dockerfile'), $dockerfile, $utf8)
}
[IO.File]::WriteAllText((Join-Path $fixture 'buildkitd.toml'), (('[registry."{0}:5000"]'+[Environment]::NewLine+'  http = true'+[Environment]::NewLine) -f $registry), $utf8)
[IO.File]::WriteAllText((Join-Path $fixture 'registries.conf'), (('[[registry]]'+[Environment]::NewLine+'location = "{0}:5000"'+[Environment]::NewLine+'insecure = true'+[Environment]::NewLine) -f $registry), $utf8)
try {
  if ($Jar) {
    docker run --rm --name $compiler --user 0 --workdir /workspace --mount "type=bind,source=$fixture,target=/workspace" -e 'CACHE_DIR=/workspace/.cache/maven' $MavenImage /bin/sh -c 'set -eu; mvn -B -ntp -Dmaven.repo.local="$CACHE_DIR" clean package'
    if ($LASTEXITCODE -ne 0 -or !(Test-Path (Join-Path $fixture 'target/ad-gateway-1.0.0.jar'))) { throw 'Maven compilation failed to produce the Dockerfile COPY source' }
  }
  docker network create $network | Out-Null
  if ($LASTEXITCODE -ne 0) { throw 'Cannot create smoke network' }
  docker run --detach --name $registry --network $network registry:2 | Out-Null
  if ($LASTEXITCODE -ne 0) { throw 'Cannot start local test registry' }
  $dockerArgs = @('run', '--rm', '--name', $builder, '--network', $network, '--privileged', '--entrypoint', '/usr/local/bin/entrypoint.sh',
    '--mount', "type=bind,source=$fixture,target=/workspace",
    '--mount', "type=bind,source=$fixture\buildkitd.toml,target=/home/user/.config/buildkit/buildkitd.toml,readonly",
    '--mount', "type=bind,source=$fixture\registries.conf,target=/etc/containers/registries.conf,readonly",
    '-e', "IMAGE_URL=$($registry):5000/smoke:v1", '-e', 'AXIOPS_RESULT_FILE=/workspace/result.json', $Image)
  & docker @dockerArgs
  if ($LASTEXITCODE -ne 0) { throw 'Rootless build/push smoke test failed' }
  $result = Get-Content (Join-Path $fixture 'result.json') -Raw | ConvertFrom-Json
  if ($result.image_digest -notmatch '^sha256:[a-f0-9]{64}$') { throw 'Invalid manifest digest' }
  $actual = docker run --rm --network $network --entrypoint skopeo $Image inspect --tls-verify=false --format '{{.Digest}}' "docker://$($registry):5000/smoke@$($result.image_digest)"
  if ($LASTEXITCODE -ne 0 -or $actual -ne $result.image_digest) { throw 'Reported digest is not a pullable manifest' }
  if ($Jar) {
    # Export exactly the pushed digest and run its Dockerfile ENTRYPOINT.
    docker run --rm --network $network --mount "type=bind,source=$fixture,target=/workspace" --entrypoint skopeo $Image copy --src-tls-verify=false "docker://$($registry):5000/smoke@$($result.image_digest)" "docker-archive:/workspace/runtime.tar:axiops/jar-smoke:$suffix"
    if ($LASTEXITCODE -ne 0) { throw 'Cannot export pushed JAR image' }
    docker load --input (Join-Path $fixture 'runtime.tar') | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Cannot load pushed JAR image' }
    $output = docker run --rm "axiops/jar-smoke:$suffix"
    if ($LASTEXITCODE -ne 0 -or $output -notcontains 'AXIOPS_JAR_SMOKE_OK') { throw 'Built JAR image ENTRYPOINT did not execute successfully' }
    Write-Host 'PASS: Maven package -> single-stage Dockerfile -> push -> execute image by verified digest'
  }
  Write-Host "PASS: rootless build, push, and pullable manifest digest $actual"
} finally {
  $ErrorActionPreference = 'Continue'
  docker rm -f $compiler $builder $registry 2>$null | Out-Null
  if ($Jar) { docker image rm "axiops/jar-smoke:$suffix" 2>$null | Out-Null }
  docker network rm $network 2>$null | Out-Null
  Write-Host "Smoke test fixture: $fixture"
}
