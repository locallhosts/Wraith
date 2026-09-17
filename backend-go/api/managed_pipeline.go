package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/locallhosts/Wraith/backend-go/orchestrator"
	"github.com/locallhosts/Wraith/backend-go/store"
)

var pipelineStageRank = map[string]int{"lint":10,"provision":20,"simulate":30,"validate":40,"soar":50,"done":60,"failed":100}

func ManagedPythonPipelineTrigger(s *Server, configuredES, configuredNeo4j string) func(string,string) {
	return func(runID, rulePath string) {
		ctx:=context.Background(); started:=time.Now()
		update:=func(stage string,passed *bool,reason string){
			run,err:=s.Store.GetRun(ctx,runID);if err!=nil{return}
			current:=pipelineStageRank[run.Stage];next:=pipelineStageRank[stage]
			if stage!="failed" && next<current{return}
			if stage==run.Stage && reason==run.Reason && passed==nil{return}
			run.Stage=stage;run.Passed=passed;run.Reason=reason
			if err:=s.Store.PutRun(ctx,run);err==nil{level:="info";if stage=="failed"{level="error"};_ = s.Store.AppendRunEvent(ctx,&store.RunEvent{RunID:runID,Stage:stage,Level:level,Message:reason})}
		}
		update("provision",nil,"preparing validation environment")
		esAddr,neo4jAddr:=configuredES,configuredNeo4j
		if envBool("WRAITH_EPHEMERAL_ENV",false){
			cli,err:=orchestrator.New();if err!=nil{failRun(s,ctx,runID,fmt.Sprintf("docker client: %v",err),started);return}
			env,err:=orchestrator.Provision(ctx,cli,runID);if err!=nil{_ = cli.Close();failRun(s,ctx,runID,fmt.Sprintf("provisioning validation environment: %v",err),started);return}
			defer func(){if err:=orchestrator.Teardown(context.Background(),cli,env);err!=nil&&s.Log!=nil{s.Log.Error("validation environment cleanup failed","run_id",runID,"error",err);_ = s.Store.AppendRunEvent(context.Background(),&store.RunEvent{RunID:runID,Stage:"failed",Level:"error",Message:"environment cleanup: "+err.Error()})};_ = cli.Close()}()
			esAddr="http://127.0.0.1:"+env.ElasticPort;neo4jAddr="bolt://127.0.0.1:"+env.Neo4jBoltPort
		}else if esAddr==""||neo4jAddr==""{failRun(s,ctx,runID,"validation service addresses are not configured",started);return}
		update("simulate",nil,"validation environment ready; starting Python engine")
		runCtx,cancel:=context.WithTimeout(ctx,15*time.Minute);defer cancel()
		cmd:=exec.CommandContext(runCtx,"python3","engine-python/run_pipeline.py","--rule",rulePath,"--run-id",runID,"--es-addr",esAddr,"--neo4j-addr",neo4jAddr,"--output-dir",s.outputDir(),"--open-pr")
		stdout,err:=cmd.StdoutPipe();if err!=nil{failRun(s,ctx,runID,fmt.Sprintf("create stdout pipe: %v",err),started);return}
		stderr,err:=cmd.StderrPipe();if err!=nil{failRun(s,ctx,runID,fmt.Sprintf("create stderr pipe: %v",err),started);return}
		if err:=cmd.Start();err!=nil{failRun(s,ctx,runID,fmt.Sprintf("start Python engine: %v",err),started);return}
		var wg sync.WaitGroup;wg.Add(2)
		go func(){defer wg.Done();streamPipelineOutput(stdout,func(line string){if s.Log!=nil{s.Log.Info("pipeline","run_id",runID,"stream","stdout","line",line)};updateStageFromLine(update,line)})}()
		go func(){defer wg.Done();streamPipelineOutput(stderr,func(line string){if s.Log!=nil{s.Log.Warn("pipeline","run_id",runID,"stream","stderr","line",line)};updateStageFromLine(update,line)})}()
		runErr:=cmd.Wait();wg.Wait()
		if runCtx.Err()==context.DeadlineExceeded{failRun(s,ctx,runID,"Python pipeline exceeded 15 minute control-plane timeout",started);return}
		reportPassed,reportReason:=readPipelineVerdict(s.outputDir(),runID)
		if runErr!=nil{reason:=runErr.Error();if reportReason!=""{reason=reportReason+"; "+reason};failRun(s,ctx,runID,reason,started);return}
		if reportPassed==nil{failRun(s,ctx,runID,"pipeline exited successfully but produced no boolean report verdict",started);return}
		if *reportPassed{update("done",reportPassed,fmt.Sprintf("pipeline completed in %s",time.Since(started).Round(time.Second)))}else{update("failed",reportPassed,reportReason)}
	}
}

func updateStageFromLine(update func(string,*bool,string),line string){
	lower:=strings.ToLower(line)
	switch{case strings.Contains(lower,"soar"):update("soar",nil,line);case strings.Contains(lower,"validat"):update("validate",nil,line);case strings.Contains(lower,"robust"),strings.Contains(lower,"mutation"),strings.Contains(lower,"quality"):update("validate",nil,line);case strings.Contains(lower,"attack simulation"),strings.Contains(lower,"attack graph"):update("simulate",nil,line);case strings.Contains(lower,"baseline"):update("simulate",nil,line);case strings.Contains(lower,"translat"):update("lint",nil,line)}
}
func streamPipelineOutput(r io.Reader,consume func(string)){scanner:=bufio.NewScanner(r);scanner.Buffer(make([]byte,64*1024),1024*1024);for scanner.Scan(){consume(scanner.Text())}}

func readPipelineVerdict(outputDir,runID string)(*bool,string){
	data,err:=os.ReadFile(fmt.Sprintf("%s/%s/report.json",outputDir,runID));if err!=nil{return nil,""}
	var report struct{Passed bool `json:"passed"`;Reason string `json:"reason"`;Error string `json:"error"`}
	if err:=json.Unmarshal(data,&report);err!=nil{return nil,"malformed report.json: "+err.Error()}
	reason:=report.Reason;if reason==""{reason=report.Error};return &report.Passed,reason
}
func failRun(s *Server,ctx context.Context,runID,reason string,started time.Time){passed:=false;if run,err:=s.Store.GetRun(ctx,runID);err==nil{run.Stage="failed";run.Passed=&passed;run.Reason=reason;_ = s.Store.PutRun(ctx,run);_ = s.Store.AppendRunEvent(ctx,&store.RunEvent{RunID:runID,Stage:"failed",Level:"error",Message:reason})};if s.Log!=nil{s.Log.Error("pipeline failed","run_id",runID,"duration",time.Since(started).Round(time.Second),"error",reason)}}
func envBool(key string,fallback bool)bool{v:=strings.TrimSpace(strings.ToLower(os.Getenv(key)));switch v{case "1","true","yes","on":return true;case "0","false","no","off":return false;default:return fallback}}
