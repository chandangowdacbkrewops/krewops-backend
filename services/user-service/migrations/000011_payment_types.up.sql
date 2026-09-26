CREATE TABLE payment_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(80) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payment_types_active
    ON payment_types(is_active);

CREATE TABLE work_category_payment_types (
    work_category_id UUID NOT NULL
        REFERENCES work_categories(id)
        ON DELETE CASCADE,
    payment_type_id UUID NOT NULL
        REFERENCES payment_types(id)
        ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_work_category_payment_type
        UNIQUE (work_category_id, payment_type_id)
);

CREATE INDEX idx_work_category_payment_types_category
    ON work_category_payment_types(work_category_id);

CREATE TABLE work_type_payment_types (
    work_type_id UUID NOT NULL
        REFERENCES work_types(id)
        ON DELETE CASCADE,
    payment_type_id UUID NOT NULL
        REFERENCES payment_types(id)
        ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_work_type_payment_type
        UNIQUE (work_type_id, payment_type_id)
);

CREATE INDEX idx_work_type_payment_types_work_type
    ON work_type_payment_types(work_type_id);

INSERT INTO payment_types (code, name)
VALUES
    ('hourly', 'Hourly'),
    ('per_day', 'Per day'),
    ('fixed', 'Fixed')
ON CONFLICT (code) DO NOTHING;

INSERT INTO work_category_payment_types (work_category_id, payment_type_id)
SELECT wc.id, pt.id
FROM work_categories wc
CROSS JOIN payment_types pt
WHERE pt.code IN ('hourly', 'per_day', 'fixed')
ON CONFLICT (work_category_id, payment_type_id) DO NOTHING;
