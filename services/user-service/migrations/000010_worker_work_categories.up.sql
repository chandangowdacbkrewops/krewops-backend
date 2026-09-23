-- CreateWorkerProfile now captures a single work_category selection
-- instead of a list of specific work_type_ids. Introduces the join table
-- backing that relationship.

CREATE TABLE worker_work_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    worker_profile_id UUID NOT NULL
        REFERENCES worker_profiles(id)
        ON DELETE CASCADE,

    work_category_id UUID NOT NULL
        REFERENCES work_categories(id)
        ON DELETE RESTRICT,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_worker_work_category
        UNIQUE (worker_profile_id, work_category_id)
);

CREATE INDEX idx_worker_work_categories_worker
    ON worker_work_categories(worker_profile_id);

CREATE INDEX idx_worker_work_categories_category
    ON worker_work_categories(work_category_id);

-- Backfill each existing worker's single work category from the category
-- of their previously selected primary (or earliest) work type, preserving
-- continuity across the work_type -> work_category shift on
-- CreateWorkerProfile.
INSERT INTO worker_work_categories (worker_profile_id, work_category_id)
SELECT DISTINCT ON (wwt.worker_profile_id)
    wwt.worker_profile_id, wt.category_id
FROM worker_work_types wwt
JOIN work_types wt ON wt.id = wwt.work_type_id
WHERE wwt.is_active = TRUE
ORDER BY wwt.worker_profile_id, wwt.is_primary DESC, wwt.created_at ASC
ON CONFLICT (worker_profile_id, work_category_id) DO NOTHING;
