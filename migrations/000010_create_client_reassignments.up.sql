-- 000010_create_client_reassignments.up.sql

CREATE TABLE IF NOT EXISTS client_reassignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    previous_seller_id UUID NOT NULL REFERENCES users(id),
    new_seller_id UUID NOT NULL REFERENCES users(id),
    reassigned_by UUID NOT NULL REFERENCES users(id),
    reason TEXT,
    inactive_days_at_reassignment INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_client_reassignments_client_id ON client_reassignments(client_id);
CREATE INDEX IF NOT EXISTS idx_client_reassignments_new_seller_id ON client_reassignments(new_seller_id);
CREATE INDEX IF NOT EXISTS idx_client_reassignments_created_at ON client_reassignments(created_at);
