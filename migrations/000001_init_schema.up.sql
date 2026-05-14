-- 000001_init_schema.up.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    branch_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE visits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    salesperson_id UUID NOT NULL REFERENCES users(id),
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    date DATE NOT NULL,
    client_name VARCHAR(255) NOT NULL,
    client_cnpj VARCHAR(20) NOT NULL,
    client_email VARCHAR(255),
    contact_person VARCHAR(255),
    contact_phone VARCHAR(20),
    branch_phone VARCHAR(20),
    address TEXT NOT NULL,
    subject TEXT NOT NULL,
    conclusion TEXT,
    arrival_time TIMESTAMP WITH TIME ZONE,
    departure_time TIMESTAMP WITH TIME ZONE,
    km_start DOUBLE PRECISION,
    km_end DOUBLE PRECISION,
    observations TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE visit_photos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    visit_id UUID NOT NULL REFERENCES visits(id) ON DELETE CASCADE,
    bucket_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_visits_salesperson ON visits(salesperson_id);
CREATE INDEX idx_visits_status ON visits(status);
CREATE INDEX idx_visits_date ON visits(date);
