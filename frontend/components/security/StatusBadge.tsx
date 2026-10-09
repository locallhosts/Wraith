export default function StatusBadge({ status }: { status: string }) {
  const value = status.toLowerCase();
  const tone = value.includes("fail") || value.includes("reject") ? "danger" : value.includes("pass") || value.includes("online") || value.includes("complete") ? "success" : "warning";
  return <span className={`status-badge ${tone}`}><i />{status}</span>;
}
