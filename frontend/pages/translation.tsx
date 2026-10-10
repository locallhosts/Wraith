import { useRouter } from "next/router";
import useSWR from "swr";
import PlatformNav from "../components/PlatformNav";
import { fetchRunTranslation } from "../lib/api";

export default function TranslationPage() {
  const router = useRouter();
  const runId =
    typeof router.query.runId === "string" ? router.query.runId : "";

  const { data, error, isLoading } = useSWR(
    runId ? ["run-translation", runId] : null,
    () => fetchRunTranslation(runId),
  );

  return (
    <div className="wraith-shell">
      <PlatformNav />
      <main className="wraith-main">
        <section className="wraith-page-header">
          <div>
            <div className="wraith-eyebrow">Detection Engineering</div>
            <h1>Sigma Translation</h1>
            <p>
              Inspect the Elasticsearch Query DSL generated from a real Wraith
              validation run.
            </p>
          </div>
        </section>

        {!runId && (
          <div className="wraith-panel">
            <h2>Missing run</h2>
            <p>Select a validation run to inspect its translated query.</p>
          </div>
        )}

        {isLoading && (
          <div className="wraith-panel">
            <p>Loading translated query…</p>
          </div>
        )}

        {error && (
          <div className="wraith-panel">
            <h2>Translation unavailable</h2>
            <p>{error.message}</p>
          </div>
        )}

        {data && (
          <>
            <div className="wraith-stat-grid">
              <div className="wraith-panel">
                <div className="wraith-eyebrow">Run</div>
                <strong>{data.run_id}</strong>
              </div>
              <div className="wraith-panel">
                <div className="wraith-eyebrow">Status</div>
                <strong>Translated</strong>
              </div>
              <div className="wraith-panel">
                <div className="wraith-eyebrow">Backend</div>
                <strong>Elasticsearch</strong>
              </div>
            </div>

            <section className="wraith-panel">
              <div className="wraith-panel-header">
                <div>
                  <div className="wraith-eyebrow">Generated query</div>
                  <h2>Elasticsearch Query DSL</h2>
                </div>
                <span className="wraith-badge">Read-only</span>
              </div>
              <pre className="wraith-code">
                {JSON.stringify(data.query_dsl, null, 2)}
              </pre>
            </section>

            <section className="wraith-panel">
              <div className="wraith-eyebrow">Artifact</div>
              <code>{data.path}</code>
            </section>
          </>
        )}
      </main>
    </div>
  );
}
