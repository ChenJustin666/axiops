-- 修复 admin 用户密码为 REPLACE_ME 的正确 bcrypt hash
-- 在已有数据库上执行此脚本修复密码

UPDATE `user` SET `password` = '$2a$10$nnF1vx6Uh1ptk.IhwiwyHeOXDpMgw8hYzXEd.C/TFRYY9/7.093MK' WHERE `username` = 'admin';
