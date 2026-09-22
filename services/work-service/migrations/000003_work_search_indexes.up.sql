CREATE INDEX idx_work_postings_work_type_id
ON work_postings(work_type_id);

CREATE INDEX idx_work_postings_city
ON work_postings(LOWER(city));

CREATE INDEX idx_work_postings_state
ON work_postings(LOWER(state));
