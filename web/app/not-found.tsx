import Link from "next/link";

export default function NotFound() {
  return (
    <div className="hero">
      <h1>Página não encontrada</h1>
      <p className="lead">Não achamos o que você procurava. O endereço pode estar errado ou o registro não existir.</p>
      <p>
        <Link className="button" href="/">Voltar ao início</Link>
      </p>
    </div>
  );
}
