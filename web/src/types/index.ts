export enum Status {
  Created = "created",
  Pending = "pending",
  Started = "started",
  Succeeded = "succeeded",
  Failed = "failed",
  Errored = "errored",
  Aborted = "aborted",
  Skipped = "skipped",
}

export interface Workflow {
  id: string;
  name: string;
  config: any;
  currBuild?: Build;
}

export interface Build {
  id: string;
  workflowId: string;
  status: Status;
}

export interface Job {
  id: string;
  buildId: string;
  name: string;
  status: Status;
}

export interface Task {
  id: string;
  jobId: string;
  name: string;
  status: Status;
}

export interface LogMessage {
  msg: string;
  time: string;
}
