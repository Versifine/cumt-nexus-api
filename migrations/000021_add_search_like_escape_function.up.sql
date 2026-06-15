CREATE OR REPLACE FUNCTION escape_like_query(raw text)
RETURNS text
LANGUAGE sql
IMMUTABLE
STRICT
AS $$
    SELECT replace(replace(replace(raw, '\', '\\'), '%', '\%'), '_', '\_')
$$;
