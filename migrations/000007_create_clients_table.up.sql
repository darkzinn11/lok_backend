-- 000007_create_clients_table.up.sql

CREATE TABLE IF NOT EXISTS clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    branch_id UUID NOT NULL REFERENCES branches(id),
    seller_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    cnpj VARCHAR(20) NOT NULL,
    email VARCHAR(255),
    contact_person VARCHAR(255),
    contact_phone VARCHAR(20),
    fixed_phone VARCHAR(20),
    address JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_clients_branch_id ON clients(branch_id);
CREATE INDEX IF NOT EXISTS idx_clients_seller_id ON clients(seller_id);
CREATE INDEX IF NOT EXISTS idx_clients_cnpj ON clients(cnpj);
CREATE INDEX IF NOT EXISTS idx_clients_name ON clients(name);
