INSERT INTO work_types (name) VALUES
    ('Electrician'),
    ('Plumber'),
    ('Carpenter'),
    ('Painter'),
    ('Mason'),
    ('Welder'),
    ('HVAC Technician'),
    ('Driver'),
    ('Security Guard'),
    ('Housekeeping Staff'),
    ('General Labourer'),
    ('Other')
ON CONFLICT (name) DO NOTHING;
