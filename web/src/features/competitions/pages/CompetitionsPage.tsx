
export function CompetitionsPage() {
  return (
    <section className="space-y-8">
      <div className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-widest text-sofi-blue">
          Observatório do Futebol
        </p>

        <h1 className="font-heading text-4xl font-bold tracking-tight text-sofi-midnight">
          Competições
        </h1>

        <p className="max-w-2xl text-base leading-relaxed text-gray-600">
          Dados financeiros e informações públicas das
          competições do futebol brasileiro.
        </p>
      </div>

      <div className="rounded-2xl border border-dashed border-gray-300 bg-white/60 p-12 text-center">
        <h2 className="font-heading text-lg font-semibold text-sofi-midnight">
          Competições
        </h2>

        <p className="mt-2 text-sm text-gray-500">
          Os campeonatos disponíveis serão apresentados aqui.
        </p>
      </div>
    </section>
  )
}
