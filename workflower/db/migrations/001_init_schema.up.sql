CREATE SCHEMA IF NOT EXISTS minici;

SET SEARCH_PATH TO minici, PUBLIC;

CREATE TABLE IF NOT EXISTS workflows (
  id      uuid  NOT NULL,
  name    text  NOT NULL UNIQUE,
  config  jsonb NOT NULL,
  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS builds (
  id          uuid        NOT NULL,
  workflow_id uuid        NOT NULL,
  status      text        NOT NULL,
  plan        jsonb       NOT NULL,
  started_at  timestamp   DEFAULT NULL,
  finished_at timestamp   DEFAULT NULL,
  PRIMARY KEY (id),
  FOREIGN KEY (workflow_id) REFERENCES workflows (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tasks (
  id        uuid  NOT NULL,
  build_id  uuid  NOT NULL,
  status    text  NOT NULL,
  step      jsonb NOT NULL,
  PRIMARY KEY (id),
  FOREIGN KEY (build_id) REFERENCES builds (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS task_events (
  task_id     uuid      NOT NULL,
  event_type  text      NOT NULL,        
  occured_at  timestamp NOT NULL,
  payload     jsonb     NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE
);
