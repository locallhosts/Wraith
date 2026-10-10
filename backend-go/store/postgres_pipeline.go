package store

import ("context"; "database/sql"; "errors"; "fmt"; "time")

const pipelineSchemaSQL = `
CREATE TABLE IF NOT EXISTS run_events (
 id BIGSERIAL PRIMARY KEY,
 run_id TEXT NOT NULL REFERENCES runs(run_id) ON DELETE CASCADE,
 stage TEXT NOT NULL,
 level TEXT NOT NULL,
 message TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_run_events_run_id_id ON run_events(run_id,id DESC);
CREATE TABLE IF NOT EXISTS pipeline_jobs (
 id BIGSERIAL PRIMARY KEY,
 run_id TEXT NOT NULL UNIQUE REFERENCES runs(run_id) ON DELETE CASCADE,
 rule_path TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('queued','running','succeeded','failed','cancelled')),
 attempts INTEGER NOT NULL DEFAULT 0,
 max_attempts INTEGER NOT NULL DEFAULT 3,
 available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 locked_by TEXT,
 locked_at TIMESTAMPTZ,
 last_error TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_pipeline_jobs_claim ON pipeline_jobs(status,available_at,id);
`

func (p *PostgresStore) applyPipelineSchema(ctx context.Context) error { _,err:=p.db.ExecContext(ctx,pipelineSchemaSQL); return err }

func (p *PostgresStore) AppendRunEvent(ctx context.Context,e *RunEvent)error{return p.db.QueryRowContext(ctx,`INSERT INTO run_events(run_id,stage,level,message) VALUES($1,$2,$3,$4) RETURNING id,created_at`).Scan(&e.ID,&e.CreatedAt)}
func (p *PostgresStore) ListRunEvents(ctx context.Context,runID string,limit int)([]*RunEvent,error){if limit<=0{limit=500};rows,err:=p.db.QueryContext(ctx,`SELECT id,run_id,stage,level,message,created_at FROM run_events WHERE run_id=$1 ORDER BY id DESC LIMIT $2`,runID,limit);if err!=nil{return nil,err};defer rows.Close();var out []*RunEvent;for rows.Next(){var e RunEvent;if err:=rows.Scan(&e.ID,&e.RunID,&e.Stage,&e.Level,&e.Message,&e.CreatedAt);err!=nil{return nil,err};out=append(out,&e)};return out,rows.Err()}
func(p *PostgresStore)EnqueuePipelineJob(ctx context.Context,j *PipelineJob)error{if j.MaxAttempts<=0{j.MaxAttempts=3};if j.AvailableAt.IsZero(){j.AvailableAt=time.Now()};_,err:=p.db.ExecContext(ctx,`INSERT INTO pipeline_jobs(run_id,rule_path,status,attempts,max_attempts,available_at,last_error) VALUES($1,$2,'queued',0,$3,$4,'') ON CONFLICT(run_id) DO UPDATE SET rule_path=EXCLUDED.rule_path,status='queued',attempts=0,max_attempts=EXCLUDED.max_attempts,available_at=EXCLUDED.available_at,last_error='',locked_by=NULL,locked_at=NULL,updated_at=now() WHERE pipeline_jobs.status IN ('failed','cancelled')`,j.RunID,j.RulePath,j.MaxAttempts,j.AvailableAt);return err}
func(p *PostgresStore)GetPipelineJob(ctx context.Context,id int64)(*PipelineJob,error){var j PipelineJob;var locked sql.NullTime;err:=p.db.QueryRowContext(ctx,`SELECT id,run_id,rule_path,status,attempts,max_attempts,available_at,COALESCE(locked_by,''),locked_at,COALESCE(last_error,''),created_at,updated_at FROM pipeline_jobs WHERE id=$1`,id).Scan(&j.ID,&j.RunID,&j.RulePath,&j.Status,&j.Attempts,&j.MaxAttempts,&j.AvailableAt,&j.LockedBy,&locked,&j.LastError,&j.CreatedAt,&j.UpdatedAt);if errors.Is(err,sql.ErrNoRows){return nil,ErrNotFound};if locked.Valid{j.LockedAt=&locked.Time};return &j,err}
func(p *PostgresStore)ListPipelineJobs(ctx context.Context,limit int)([]*PipelineJob,error){if limit<=0{limit=200};rows,err:=p.db.QueryContext(ctx,`SELECT id,run_id,rule_path,status,attempts,max_attempts,available_at,COALESCE(locked_by,''),locked_at,COALESCE(last_error,''),created_at,updated_at FROM pipeline_jobs ORDER BY created_at DESC LIMIT $1`,limit);if err!=nil{return nil,err};defer rows.Close();var out []*PipelineJob;for rows.Next(){var j PipelineJob;var locked sql.NullTime;if err:=rows.Scan(&j.ID,&j.RunID,&j.RulePath,&j.Status,&j.Attempts,&j.MaxAttempts,&j.AvailableAt,&j.LockedBy,&locked,&j.LastError,&j.CreatedAt,&j.UpdatedAt);err!=nil{return nil,err};if locked.Valid{j.LockedAt=&locked.Time};out=append(out,&j)};return out,rows.Err()}
func(p *PostgresStore)RetryPipelineJob(ctx context.Context,id int64)error{res,err:=p.db.ExecContext(ctx,`UPDATE pipeline_jobs SET status='queued',attempts=0,last_error='',available_at=now(),locked_by=NULL,locked_at=NULL,updated_at=now() WHERE id=$1`,id);if err!=nil{return err};n,_:=res.RowsAffected();if n==0{return ErrNotFound};return nil}
func(p *PostgresStore)ClaimPipelineJob(ctx context.Context,workerID string,lease time.Duration)(*PipelineJob,error){tx,err:=p.db.BeginTx(ctx,nil);if err!=nil{return nil,err};defer tx.Rollback();var j PipelineJob;var locked time.Time;err=tx.QueryRowContext(ctx,`SELECT id,run_id,rule_path,status,attempts,max_attempts,available_at,COALESCE(locked_by,''),COALESCE(locked_at,now()),COALESCE(last_error,''),created_at,updated_at FROM pipeline_jobs WHERE (status='queued' AND available_at<=now()) OR (status='running' AND locked_at<now()-($1::interval)) ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1`,fmt.Sprintf("%d seconds",int(lease.Seconds()))).Scan(&j.ID,&j.RunID,&j.RulePath,&j.Status,&j.Attempts,&j.MaxAttempts,&j.AvailableAt,&j.LockedBy,&locked,&j.LastError,&j.CreatedAt,&j.UpdatedAt);if errors.Is(err,sql.ErrNoRows){return nil,ErrNoJobAvailable};if err!=nil{return nil,err};j.Attempts++;j.Status="running";j.LockedBy=workerID;locked=time.Now();j.LockedAt=&locked;if _,err=tx.ExecContext(ctx,`UPDATE pipeline_jobs SET status='running',attempts=$2,locked_by=$3,locked_at=$4,updated_at=now() WHERE id=$1`,j.ID,j.Attempts,workerID,locked);err!=nil{return nil,err};if err=tx.Commit();err!=nil{return nil,err};return &j,nil}
func(p *PostgresStore)CompletePipelineJob(ctx context.Context,id int64,status,lastError string)error{res,err:=p.db.ExecContext(ctx,`UPDATE pipeline_jobs SET status=$2,last_error=$3,locked_by=NULL,locked_at=NULL,updated_at=now() WHERE id=$1`,id,status,lastError);if err!=nil{return err};n,_:=res.RowsAffected();if n==0{return ErrNotFound};return nil}
func(p *PostgresStore)RequeuePipelineJob(ctx context.Context,id int64,delay time.Duration,lastError string)error{res,err:=p.db.ExecContext(ctx,`UPDATE pipeline_jobs SET status='queued',last_error=$2,available_at=now()+$3::interval,locked_by=NULL,locked_at=NULL,updated_at=now() WHERE id=$1`,id,lastError,fmt.Sprintf("%d seconds",int(delay.Seconds())));if err!=nil{return err};n,_:=res.RowsAffected();if n==0{return ErrNotFound};return nil}
func(p *PostgresStore)CancelPipelineJob(ctx context.Context,runID string)error{_,err:=p.db.ExecContext(ctx,`UPDATE pipeline_jobs SET status='cancelled',last_error='cancelled by operator',locked_by=NULL,locked_at=NULL,updated_at=now() WHERE run_id=$1 AND status IN ('queued','running')`,runID);return err}
