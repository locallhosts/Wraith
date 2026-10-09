const stages = ["Rule intake", "Lint & translate", "Attack simulation", "SIEM validation", "Mutation testing", "Quality gate"];

export default function Pipeline({ active = 3 }: { active?: number }) {
  return (
    <div className="pipeline">
      {stages.map((stage, index) => {
        const state = index < active ? "complete" : index === active ? "active" : "pending";
        return (
          <div className="pipeline-step" key={stage}>
            <div className={`pipeline-node ${state}`}>{index < active ? "✓" : index + 1}</div>
            <div><strong>{stage}</strong><span>{state === "complete" ? "Completed" : state === "active" ? "Processing" : "Queued"}</span></div>
            {index !== stages.length - 1 && <div className={`pipeline-line ${index < active ? "complete" : ""}`} />}
          </div>
        );
      })}
    </div>
  );
}
