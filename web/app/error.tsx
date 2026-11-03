"use client";

export default function Erro({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="hero">
      <h1>Algo deu errado</h1>
      <p className="lead">Não conseguimos carregar os dados agora. Tente novamente em instantes.</p>
      <p>
        <button type="button" onClick={reset}>Tentar de novo</button>
      </p>
    </div>
  );
}
