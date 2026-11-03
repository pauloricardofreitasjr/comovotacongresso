# Pendências do projeto

Estado em 2026-11-03, após as Fases 0 a 4 e a revisão do site. Itens ordenados pelo que impede o lançamento.

## Bloqueiam o lançamento

| Pendência | Quem resolve |
|---|---|
| **Fase 6, o que falta:** escolher a hospedagem, apontar DNS, fazer o primeiro deploy real e testar HTTPS (o Caddy só foi validado em configuração e rate limit, sem certificado real), cópia dos backups para fora da máquina e monitor externo. O restante está pronto em `deploy/` e `docs/operacao.md` | Responsável decide a hospedagem; Claude executa |
| **Licença das fotos:** pedir confirmação por escrito à Câmara (e ao Senado na v1.1). Ver `discovery/README.md` §3.3 | Responsável |
| **Fotos copiadas para o nosso storage:** a decisão foi copiar, mas o site hoje **carrega direto do portal da Câmara**; a cópia não foi implementada | Claude, depois da licença |
| **Crédito de fonte nas fotos:** implementado com o texto da pesquisa (rodapé e perfis). Falta só trocar pelo texto que a Câmara indicar, se ela indicar outro | Responsável, junto com a resposta da Câmara |
| **Metodologia:** revisar as seções novas (4 e 4.1) e preencher os colchetes de `metodologia.md`: canal de contato, histórico de versões, crédito das fotos | Responsável |
| **Decidir indexação:** hoje o `robots.txt` permite tudo | Responsável |

## Sensível ao tempo

- **58ª legislatura (1º/fev/2027): código pronto, falta validar na virada.** O ETL descobre a vigente por `/legislaturas` e o exercício é substituído só dentro do período de cada legislatura. Não verificado, porque a API ainda não publica a 58: se `/legislaturas` e `/deputados?idLegislatura=58` aparecem completos no dia, e se a fonte emite `FIM_MANDATO` dos não reeleitos da 57 (na transição 56→57 emitiu em 31/01/2023). Rodar `sync-parlamentares` nos primeiros dias de fevereiro e conferir `em_exercicio`.
- O texto "desde fevereiro de 2023" no site continua correto para a série que cobre as duas legislaturas; rever se o responsável decidir mostrar só a vigente.

## Dados a validar ou decidir

- Por que líderes como Antonio Brito têm 100% de presença em sessão e 35% de voto nas nominais. São medidas diferentes (explicado em `discovery/README.md` §3.5), mas a **causa não foi investigada**.
- **Listas de votos duplicadas na fonte (tratado, falta a Câmara confirmar):** a lista de votos da `2399598-70` (PL 5230/2023, Emenda 40) é idêntica à da `2269745-90`. O ETL passou a detectar listas idênticas (≥ 20 votos) e a pôr em quarentena: lista trocada → votos ocultos e votação fora das métricas; mesmo ato registrado duas vezes → conta uma só. Hoje são 4 votações (1 e 3). Se a Câmara corrigir, a marca some sozinha no ciclo seguinte. A causa é hipótese até a Câmara confirmar; o relato está em `docs/comunicacoes/relato_inconsistencias_camara.md`, ainda não enviado. Também sem resposta: MPV 1151/2022 (total da descrição ≠ total da lista; só avisamos).
- 887 votos sem posição na fonte (2 votações, `tipoVoto` nulo) contam como participação. Confirmar que é isso mesmo.
- 1.650 vínculos "prováveis" com a proposição, quase todos de votações de procedimento. Amostrar para avaliar a qualidade da inferência.
- A heurística da "votação principal" foi validada em um único exemplo.
- Ausência justificada: a fonte só lista quem registrou presença.
- Nomes de bancada truncados pela própria Câmara (como "Bl UniPpPsd...").

## Técnico

- **A CI nunca rodou.** O repositório não tem remoto, então os jobs de Go e do site estão escritos, mas sem execução real.
- O `sync-parlamentares` semanal roda dentro do agendador (`etl agendar`), mas só foi exercitado em testes; a primeira execução real é no próximo domingo após o deploy.
- O agendador não impede que um `etl daily` manual rode ao mesmo tempo que o agendado (as gravações são idempotentes, mas duplicam requisições à Câmara).
- `etl_errors` acumula as mesmas linhas de "indisponível" a cada execução. Falta limpeza ou deduplicação.
- Cobertura: `cmd/api` e `cmd/etl` quase sem teste (são só a fiação). `go test -race` não roda sem compilador C. O `golangci-lint` do plano não foi configurado.
- O documento de arquitetura original (`documento_de_arquitetura_e_especifica_o_t_cnica.md`) está **parcialmente defasado** em relação ao que foi implementado (nomes de métricas, SQL do §4.4).

## Melhorias do site

- Sitemap (depende da decisão de indexação). As imagens de compartilhamento (Open Graph) e o botão Compartilhar já existem para votação, deputado e lei; o sucesso do botão de copiar só foi testado em contexto automatizado.
- Teste com leitor de tela e conferência visual do modo claro (o contraste foi verificado por cálculo, não visualmente).
- Celular: as telas foram conferidas em 320, 360 e 390 px (sem rolagem horizontal), mas em quadros dentro do desktop, não em aparelho real; falta testar toque, teclado virtual e o botão de compartilhar num celular de verdade.
- Filtros recolhíveis no celular (os filtros ocupam bastante tela antes da lista).

## Fora do MVP, já registrado

- **Senado (v1.1):** votação simbólica e secreta, período de exercício dos senadores, termos das fotos.
- **Fidelidade partidária (v2):** mapa partido→bloco.
- **Público de jornalistas e pesquisadores (v2):** exportação CSV, API pública documentada.

## Ordem sugerida

A Fase 6 e a legislatura 58 são trabalho de implementação. A licença das fotos e a revisão da metodologia são do responsável e podem andar em paralelo, porque o texto de atribuição depende delas.
