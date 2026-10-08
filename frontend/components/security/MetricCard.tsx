export default function MetricCard({ label, value, detail, tone = "cyan" }: { label: string; value: string | number; detail?: string; tone?: "cyan" | "green" | "amber" | "rose" }) {
  return (
    <article className={`metric-card metric-${tone}`}>
      <div className="metric-label">{label}</div>
      <div className="metric-value">{value}</div>
      {detail && <div className="metric-detail">{detail}</div>}
    </article>
  );
}
