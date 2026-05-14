UPDATE visits
SET status = CASE status
    WHEN 'Análise' THEN 'IN_ANALYSIS'
    WHEN 'Pendente' THEN 'PENDING'
    WHEN 'Enviado' THEN 'COMPLETED'
    WHEN 'Cancelado' THEN 'CANCELED'
    ELSE status
END;

UPDATE branches
SET status = 'ACTIVE'
WHERE id <> '11111111-1111-1111-1111-111111111111';
