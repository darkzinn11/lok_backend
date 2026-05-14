-- 000003_add_must_change_password.up.sql
ALTER TABLE users ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT TRUE;

-- Para usuários existentes (como o de seed), vamos assumir que eles já trocaram para não quebrar a demo atual
UPDATE users SET must_change_password = FALSE WHERE email IN ('diretor@lokcenter.com', 'gerente@lokcenter.com');
