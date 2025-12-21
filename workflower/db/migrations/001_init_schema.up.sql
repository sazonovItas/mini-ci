CREATE SCHEMA IF NOT EXISTS minici;

SET SEARCH_PATH TO minici, PUBLIC;

CREATE TABLE IF NOT EXISTS workflows (
  id            varchar(36) NOT NULL,
  name          text        NOT NULL UNIQUE,
  curr_build_id varchar(36) NOT NULL DEFAULT 'null',
  config        jsonb       NOT NULL,
  PRIMARY KEY   (id)
);

CREATE OR REPLACE FUNCTION scheduled_at_trigger() RETURNS TRIGGER
  LANGUAGE plpgsql AS
$$ BEGIN
  NEW.scheduled_at := NOW(); 
  RETURN NEW;
END; $$;

CREATE TABLE IF NOT EXISTS builds (
  id            varchar(36) NOT NULL,
  workflow_id   varchar(36) NOT NULL DEFAULT 'null',
  status        text        NOT NULL,
  config        jsonb       DEFAULT NULL,
  plan          jsonb       DEFAULT NULL,
  created_at    timestamp   DEFAULT NOW(),
  scheduled_at  timestamp   DEFAULT NOW(),
  PRIMARY KEY (id),
  FOREIGN KEY (workflow_id) REFERENCES workflows (id) ON DELETE SET DEFAULT
);

CREATE OR REPLACE TRIGGER scheduled_at_builds_trgr
  AFTER UPDATE
  ON builds
  FOR EACH ROW EXECUTE PROCEDURE scheduled_at_trigger();

CREATE TABLE IF NOT EXISTS jobs (
  id            varchar(36) NOT NULL,
  build_id      varchar(36) NOT NULL,
  name          text        NOT NULL,
  status        text        NOT NULL,
  config        jsonb       DEFAULT NULL,
  plan          jsonb       DEFAULT NULL,
  scheduled_at  timestamp   DEFAULT NOW(),
  PRIMARY KEY (id),
  FOREIGN KEY (build_id) REFERENCES builds (id) ON DELETE CASCADE
);

CREATE OR REPLACE TRIGGER scheduled_at_jobs_trgr
  AFTER UPDATE
  ON jobs
  FOR EACH ROW EXECUTE PROCEDURE scheduled_at_trigger();

CREATE TABLE IF NOT EXISTS tasks (
  id        varchar(36) NOT NULL,
  job_id    varchar(36) NOT NULL,
  name      text        NOT NULL,
  status    text        NOT NULL,
  config    jsonb       DEFAULT NULL,
  PRIMARY KEY (id),
  FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS task_logs (
  task_id     varchar(36) NOT NULL,
  message     text        NOT NULL,
  time        timestamp   NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE
)
WITH (
  timescaledb.hypertable,
  timescaledb.partition_column='time',
  timescaledb.segmentby='task_id'
);

CREATE TABLE IF NOT EXISTS events (
  origin_id   varchar(36) NOT NULL,
  occured_at  timestamp   NOT NULL,
  event_type  text        NOT NULL,        
  payload     jsonb       NOT NULL
)
WITH (
  timescaledb.hypertable,
  timescaledb.partition_column='occured_at',
  timescaledb.segmentby='origin_id'
);
