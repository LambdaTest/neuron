ALTER TABLE repositories 
ADD job_view enum("default","ftm_only") NOT NULL DEFAULT "default";