# Como calculamos — Metodologia do Como Vota Congresso

> **RASCUNHO para revisão.** Trechos entre `[colchetes]` dependem de decisão ou confirmação. Escrito para o eleitor, em linguagem simples; a versão técnica fica em `docs/documento_de_arquitetura_e_especifica_o_t_cnica.md`.

O **Como Vota Congresso** mostra como cada deputado federal votou no Plenário da Câmara. Todos os números vêm de dados públicos e são calculados com regras fixas, as mesmas para todos os parlamentares e partidos. Esta página explica essas regras, o que elas deixam de fora e onde os números podem enganar.

## 1. De onde vêm os dados

- **Fonte:** Dados Abertos da Câmara dos Deputados (`dadosabertos.camara.leg.br`). Fonte dos dados: Câmara dos Deputados.
- **Período:** votações a partir de 1º de fevereiro de 2023 (início da 57ª Legislatura).
- **Atualização:** uma vez por dia, de madrugada. Uma votação feita hoje costuma aparecer amanhã. A data da última atualização aparece no rodapé do site.
- **Fotos:** publicadas pela Câmara dos Deputados. `[Confirmar texto de crédito após resposta formal da Câmara.]`
- Não corrigimos nem alteramos os dados. Se a Câmara corrigir um registro, a correção chega ao site no ciclo seguinte.
- **Senado:** ainda não está no site. `[Texto a ajustar quando o Senado entrar.]`

## 2. Votação nominal e votação simbólica

Nem toda votação registra o voto de cada deputado.

- **Nominal:** cada deputado vota no painel e o voto fica registrado em seu nome. Só nessas votações podemos mostrar como cada um votou.
- **Simbólica:** o presidente da sessão pergunta quem é a favor e os deputados permanecem sentados ou levantam. Não há registro individual. Mostramos o resultado (aprovada ou rejeitada) e as orientações dos partidos, mas **não** o voto de cada deputado.

Por isso, todas as taxas deste site se baseiam **somente em votações nominais**. Isso significa que elas retratam uma parte das decisões da Câmara, e não todas. Votações simbólicas costumam envolver temas de consenso ou de rotina; as nominais costumam ser as mais disputadas.

## 3. O que está incluído

- Votações do **Plenário** da Câmara (o plenário principal, com todos os deputados).
- **Fora do site:** votações em comissões.
- Votações de procedimento (requerimentos, urgências, alterações de regime) aparecem junto da matéria a que se referem. Quando não conseguimos identificar a matéria com segurança, a votação aparece em "Outras votações". Quando o vínculo é provável, mas não certo, o site avisa.

## 4. Presença em plenário

**O que mede:** em quantas votações nominais o deputado registrou voto, entre as que aconteceram enquanto ele estava em exercício.

**Como calculamos:**

> Presença (%) = votações nominais em que o deputado votou ÷ votações nominais ocorridas durante o seu exercício × 100

**Regras:**
- Contam como presença os votos **Sim, Não, Abstenção, Obstrução** e o voto do presidente da sessão (Artigo 17). Quem se abstém ou obstrui estava presente.
- Se o deputado não aparece na lista de votos de uma votação nominal, contamos como **ausente**.
- **Só contam as votações em que o deputado estava em exercício.** Períodos de licença, afastamento ou antes da posse (no caso de suplentes) não entram no cálculo, para não punir quem não era titular na data.
- Votações simbólicas não entram.

**Cuidados ao interpretar:**
- Um deputado pode estar em Brasília e não votar (por exemplo, em reunião de comissão). Mostramos o registro de voto, não o motivo da ausência.
- Ausências justificadas e não justificadas aparecem juntas. `[Avaliar se vale separar usando os dados de presença da Câmara em fase posterior.]`
- Suplentes têm poucas votações, então um percentual alto ou baixo pesa pouco. Mostramos sempre a quantidade de votações ao lado do percentual.

## 5. Alinhamento com o governo

**O que mede:** com que frequência o deputado votou como o Governo pediu.

**De onde vem o "pedido do Governo":** antes de uma votação nominal, a liderança do Governo na Câmara informa como orienta a bancada: **Sim**, **Não** ou **Liberado**. Usamos essa orientação como está registrada pela Câmara.

**Como calculamos cada voto:**

| Orientação do Governo | Voto do deputado | Resultado |
|---|---|---|
| Sim | Sim | Alinhado |
| Sim | Não, Abstenção ou Obstrução | Não alinhado |
| Não | Não | Alinhado |
| Não | Sim, Abstenção ou Obstrução | Não alinhado |
| Liberado, vazia ou inexistente | qualquer | **Não entra no cálculo** |

> Alinhamento (%) = votos alinhados ÷ votos com orientação do Governo (Sim ou Não) × 100

**Cuidados ao interpretar:**
- "Alinhado ao governo" **não** significa "a favor do Presidente". Significa que o voto coincidiu com a orientação registrada naquele dia. O Governo muda ao longo do tempo, e o cálculo sempre usa a orientação da data da votação.
- Abstenção e obstrução contam como "não alinhado" porque diferem do que o Governo pediu. É uma escolha de regra; `[detalhar o impacto no site: mostrar também quantas abstenções e obstruções houve]`.
- O voto do presidente da sessão (Artigo 17) e votos sem orientação do Governo não entram.
- Ausências não contam como "contra o governo": quem não votou fica fora do cálculo de alinhamento (e entra na presença).
- Mostramos sempre quantas votações entraram no cálculo. Percentuais com poucas votações dizem pouco.

## 6. O que não medimos

- **Fidelidade ao partido:** não calculamos neste momento. `[Previsto para versão futura.]`
- **Qualidade ou mérito do voto:** o site não diz se o voto foi bom ou ruim, nem avalia o conteúdo das leis.
- **Atuação fora do voto:** projetos de autoria, discursos, presença em comissões e emendas não entram.
- **Votações simbólicas e em comissões:** ver seções 2 e 3.

## 7. Limitações e erros

- Os dados vêm da Câmara; erros ou atrasos na fonte aparecem aqui também.
- Pequenas diferenças em relação a outros sites podem ocorrer por causa do período considerado, das votações incluídas ou do momento da atualização.
- Os cálculos são feitos por código aberto `[confirmar se o repositório será público]`; qualquer pessoa pode conferir as regras.

## 8. Contato e correções

Encontrou um erro ou tem uma dúvida? `[canal de contato: e-mail ou formulário]`. Registramos as correções e atualizações desta metodologia aqui: `[histórico de versões da página]`.

*Última revisão da metodologia: `[data]`.*
