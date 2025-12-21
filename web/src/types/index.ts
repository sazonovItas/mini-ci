enum Status {
  Created = "created",
  Pending = "pending",
  Started = "started",
  Succeeded = "succeeded",
  Failed = "failed",
  Errored = "errored",
  Aborted = "aborted",
  Skipped = "skipped",
}
export default Status;

export interface PlanNode {
  ref?: { id: string };
  next?: PlanNode;
  config?: any;
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
  plan: PlanNode; // Used to order Jobs
}

export interface Job {
  id: string;
  buildId: string;
  name: string;
  status: Status;
  plan: PlanNode; // Used to order Tasks
}

export interface Task {
  id: string;
  jobId: string;
  name: string;
  status: Status;
  config: any;
}

export interface LogMessage {
  msg: string;
  time: string;
}
