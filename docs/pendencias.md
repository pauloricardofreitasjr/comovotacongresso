# Pendências do projeto

Estado em 2026-11-03, após as Fases 0 a 4 e a revisão do site. Itens ordenados pelo que impede o lançamento.

## Bloqueiam o lançamento

| Pendência | Quem resolve |
|---|---|
| **Fase 6 inteira:** agendar o ETL, deploy, HTTPS e DNS em `www.comovotacongresso.com.br`, backups, alertas, rate limiting no proxy e senhas reais no lugar das de desenvolvimento | Claude, com decisão de hospedagem do responsável |
| **Licença das fotos:** pedir confirmação por escrito à Câmara (e ao Senado na v1.1). Ver `discovery/README.md` §3.3 | Responsável |
| **Fotos copiadas para o nosso storage:** a decisão foi copiar, mas o site hoje **carrega direto do portal da Câmara**; a cópia não foi implementada | Claude, depois da licença |
| **Crédito de fonte nas fotos** e texto definitivo de atribuição | Responsável aprova, Claude implementa |
| **Metodologia:** revisar as seções novas (4 e 4.1) e preencher os colchetes de `metodologia.md`: canal de contato, histórico de versões, se o código será aberto, crédito das fotos | Responsável |
| **Decidir indexação:** hoje o `robots.txt` permite tudo | Responsável |

## Sensível ao tempo

- **58ª legislatura (1º/fev/2027): código pronto, falta validar na virada.** O ETL descobre a vigente por `/legislaturas` e o exercício é substituído só dentro do período de cada legislatura. Não verificado, porque a API ainda não publica a 58: se `/legislaturas` e `/deputados?idLegislatura=58` aparecem completos no dia, e se a fonte emite `FIM_MANDATO` dos não reeleitos da 57 (na transição 56→57 emitiu em 31/01/2023). Rodar `sync-parlamentares` nos primeiros dias de fevereiro e conferir `em_exercicio`.
- O texto "desde fevereiro de 2023" no site continua correto para a série que cobre as duas legislaturas; rever se o responsável decidir mostrar só a vigente.

## Dados a validar ou decidir

- Por que líderes como Antonio Brito têm 100% de presença em sessão e 35% de voto nas nominais. São medidas diferentes (explicado em `discovery/README.md` §3.5), mas a **causa não foi investigada**.
- 887 votos sem posição na fonte (2 votações, `tipoVoto` nulo) contam como participação. Confirmar que é isso mesmo.
- 1.650 vínculos "prováveis" com a proposição, quase todos de votações de procedimento. Amostrar para avaliar a qualidade da inferência.
- A heurística da "votação principal" foi validada em um único exemplo.
- Ausência justificada: a fonte só lista quem registrou presença.
- Nomes de bancada truncados pela própria Câmara (como "Bl UniPpPsd...").

## Técnico

- **A CI nunca rodou.** O repositório não tem remoto, então os jobs de Go e do site estão escritos, mas sem execução real.
- O `daily` não atualiza exercício e partido dos parlamentares. Precisa de um `sync-parlamentares` periódico (por exemplo, semanal) para pegar novos suplentes, licenças e trocas de partido.
- `etl_errors` acumula as mesmas linhas de "indisponível" a cada execução. Falta limpeza ou deduplicação.
- Cobertura: `cmd/api` e `cmd/etl` quase sem teste (são só a fiação). `go test -race` não roda sem compilador C. O `golangci-lint` do plano não foi configurado.
- O documento de arquitetura original (`documento_de_arquitetura_e_especifica_o_t_cnica.md`) está **parcialmente defasado** em relação ao que foi implementado (nomes de métricas, SQL do §4.4).

## Melhorias do site

- Sitemap e imagens de compartilhamento (Open Graph) por página.
- Teste com leitor de tela e conferência visual do modo claro (o contraste foi verificado por cálculo, não visualmente).
- Filtros recolhíveis no celular.

## Fora do MVP, já registrado

- **Senado (v1.1):** votação simbólica e secreta, período de exercício dos senadores, termos das fotos.
- **Fidelidade partidária (v2):** mapa partido→bloco.
- **Público de jornalistas e pesquisadores (v2):** exportação CSV, API pública documentada.

## Ordem sugerida

A Fase 6 e a legislatura 58 são trabalho de implementação. A licença das fotos e a revisão da metodologia são do responsável e podem andar em paralelo, porque o texto de atribuição depende delas.
