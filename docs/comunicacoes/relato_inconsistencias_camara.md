# Relato de inconsistências na API de Dados Abertos (rascunho)

> **Rascunho para você enviar.** Não foi enviado a ninguém. Preencha os trechos entre `[colchetes]` e envie pelo canal de contato dos Dados Abertos (<https://dadosabertos.camara.leg.br>) ou pelo Fale Conosco do portal. Tudo abaixo foi conferido em 03/11/2026 contra a própria API e pode ser reproduzido pelos comandos da seção final. Os números e identificadores são verdadeiros; a hipótese sobre a causa está marcada como hipótese.
>
> O pedido de confirmação da licença das fotos é outra conversa (outro setor) e deve ir em mensagem separada.

---

**Assunto:** Inconsistências em `/votacoes/{id}` e `/votacoes/{id}/votos` (lista de votos duplicada entre votações)

Olá,

Sou `[seu nome]`, responsável pelo **Como Vota Congresso** (`https://www.comovotacongresso.com.br`), um site independente e sem fins lucrativos que mostra como cada deputado votou no Plenário usando a API de Dados Abertos da Câmara. O código é aberto (`https://github.com/pauloricardofreitasjr/comovotacongresso`). Ao conferir o placar escrito na descrição de cada votação nominal contra a lista de votos individuais, encontrei os casos abaixo. Antes de publicar qualquer correção do nosso lado, gostaria de saber se são erros nos dados e se haverá correção.

Das 1.079 votações nominais do Plenário que carreguei (fev/2023 a nov/2026), 1.073 trazem "Sim: N; Não: N" na descrição, e em 1.069 o placar bate exatamente com a lista de votos. Os casos que não batem:

## 1. Votação `2399598-70`: a lista de votos parece ser a de outra votação

- **O que é:** PL 5230/2023, Emenda de Plenário nº 40, sessão de 20/03/2024, registro às 22:24:43, órgão PLEN.
- **Descrição (`/votacoes/2399598-70`):** "Rejeitada a Emenda de Plenário nº 40. Sim: 35; não: 375; total: 410." (`aprovacao`: 0)
- **Votos (`/votacoes/2399598-70/votos`):** 404 registros: 315 Sim, 87 Não, 1 Abstenção, 1 Artigo 17.
- **O que observei:** essa lista de votos é **idêntica** à de `/votacoes/2269745-90` (mesmos 404 deputados com o mesmo voto, comparando o conjunto de pares deputado/voto). A votação `2269745-90` ocorreu no mesmo dia, às 19:30:58, e sua descrição é "Mantido o texto. Sim: 315; não: 87; abstenção: 1; total: 403.", ou seja, o placar dela é exatamente o da lista.
- **Hipótese (não confirmada):** a lista de votos da `2399598-70` é uma repetição da lista da `2269745-90`. Se for isso, os votos individuais atribuídos aos deputados na Emenda 40 não são os que eles registraram.
- **Pergunta:** isso é um erro? Se for, onde obtemos a lista correta, e há previsão de correção?

## 2. Votação `2345378-38`: o total da descrição e o da lista diferem

- **O que é:** MPV 1151/2022, parecer da Comissão Mista em apreciação preliminar, sessão de 30/03/2023, registro às 16:57:32.
- **Descrição (`/votacoes/2345378-38`):** "... Sim: 327; não: 27; total: 354."
- **Votos (`/votacoes/2345378-38/votos`):** 365 registros: 337 Sim, 27 Não, 1 Artigo 17.
- **Diferença:** 10 votos Sim a mais na lista e total 11 maior (365 contra 354). Não encontrei outra votação com essa lista.
- **Pergunta:** qual dos dois placares é o definitivo? Pode ser, por exemplo, registro de votos depois do fechamento do placar, mas não tenho como saber.

## 3. Pares de votações com listas de votos idênticas (possível registro duplicado)

Encontrei três pares de votações diferentes, com poucos segundos ou minutos de diferença e descrições do mesmo ato, cujas listas de votos são idênticas (mesmos deputados, mesmos votos). Aqui as descrições batem com as listas, então parece o **mesmo ato registrado duas vezes**, e não uma lista trocada:

| Votações | Horários (sessão) | Resultado na lista |
|---|---|---|
| `2414224-6` e `259094-181` | 15/12/2023, 20:47:04 e 20:47:21 (Requerimento nº 4.314/2023) | 309 Sim, 33 Não, 385 votos |
| `2420577-7` e `2417025-45` | 12/03/2024, 19:44:37 e 19:45:05 (recurso rejeitado) | 139 Sim, 290 Não, 430 votos |
| `2399598-108` e `2447901-6` | 09/07/2024, 19:26:33 e 19:36:11 (Recurso nº 18/2024) | 126 Sim, 302 Não, 429 votos |

- **Pergunta:** são mesmo a mesma deliberação registrada duas vezes (por exemplo, uma para a proposição principal e outra para o objeto)? Existe um campo que permita identificar a duplicata, para que quem consome a API não conte o mesmo voto duas vezes?

## Como reproduzir

```bash
# 1. Votação 2399598-70 e a que parece ser a origem da lista
curl -s -H 'Accept: application/json' https://dadosabertos.camara.leg.br/api/v2/votacoes/2399598-70
curl -s -H 'Accept: application/json' https://dadosabertos.camara.leg.br/api/v2/votacoes/2399598-70/votos
curl -s -H 'Accept: application/json' https://dadosabertos.camara.leg.br/api/v2/votacoes/2269745-90
curl -s -H 'Accept: application/json' https://dadosabertos.camara.leg.br/api/v2/votacoes/2269745-90/votos
# comparando os pares (deputado_.id, tipoVoto) das duas listas, o conjunto é o mesmo

# 2. MPV 1151/2022
curl -s -H 'Accept: application/json' https://dadosabertos.camara.leg.br/api/v2/votacoes/2345378-38
curl -s -H 'Accept: application/json' https://dadosabertos.camara.leg.br/api/v2/votacoes/2345378-38/votos
```

## Como o site trata isso hoje

Mostramos os votos como a API os registra e exibimos um aviso quando a descrição oficial traz um placar diferente da lista. Se vocês confirmarem os erros, podemos passar a sinalizá-los com mais clareza (ou ocultar a lista de votos individuais) até a correção. Fico à disposição para enviar a lista completa dos pares comparados.

Obrigado pela atenção e pelo trabalho com os Dados Abertos.

`[seu nome]`
`[e-mail de contato do projeto]`
