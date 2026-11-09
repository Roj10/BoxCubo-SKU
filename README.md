# BoxCubo SKU

Sistema de SKU das máquinas BoxCubo: cadastro de peças, lista de materiais por
máquina e planejamento de compra para um lote de produção.

- **backend/** — API em Go (só biblioteca padrão, sem dependências)
- **frontend/** — React + Vite
- **tools/** — dados extraídos da planilha da V2 e o script que gera o seed

## Mapa dos arquivos (para quem não é programador)

Todo arquivo de código começa com um comentário explicando, em português
simples, o que ele faz. Resumo:

| Onde | O que é |
|---|---|
| `backend/cmd/server/main.go` | Liga o servidor ("a cozinha") |
| `backend/internal/model/model.go` | O "formulário" de cada coisa: máquina, peça, fornecedor, movimentação |
| `backend/internal/store/store.go` | A "despensa": lê e grava o arquivo de dados |
| `backend/internal/store/supabase.go` | Guarda os dados no Supabase (lê ao ligar, grava só o que mudou) |
| `backend/supabase/schema.sql` | Cria as tabelas no Supabase (rodar uma vez no SQL Editor) |
| `backend/.env` | Endereço e chave secreta do Supabase (não vai para o GitHub) |
| `backend/internal/store/migracoes.go` | Atualizações automáticas de dados antigos |
| `backend/internal/api/api.go` | O "balcão": recebe os pedidos da tela (lista de peças, planejamento, cadastro) |
| `backend/internal/api/estoque.go` | Saídas, entradas e estornos |
| `backend/internal/api/minimo.go` | Estoque mínimo e reposição |
| `backend/internal/api/imagens.go` | Fotos das peças |
| `backend/internal/api/subconjuntos.go` | Subconjuntos (CPU, Shield): receita, custo, fabricação, em quais máquinas a peça vai |
| `backend/internal/api/fornecedores.go` | Página de fornecedores: resumo, peças, código do item, renomear/juntar |
| `backend/internal/api/*_test.go` | Testes automáticos (`go test ./...`) |
| `backend/data/` | **Os dados de vocês** (`boxcubo.json`) e as fotos |
| `frontend/src/App.jsx` | A tela inteira: barra lateral + área principal |
| `frontend/src/api.js` | O "garçom": leva os pedidos da tela para o backend |
| `frontend/src/planilha.js` | Monta as planilhas do Excel (.xlsx) que o sistema baixa |
| `frontend/src/components/` | Cada pedaço da tela (tabela, janelas, barra lateral…) |
| `frontend/src/styles.css` | O visual (cores, fontes) |
| `tools/` | Dados extraídos da planilha e o gerador dos dados iniciais |
| `tools/controle/` | Receitas e materiais lidos do Controle BoxCubo + o importador |

## Como rodar (Windows)

Pré-requisitos: [Go 1.22+](https://go.dev/dl/) e [Node.js 18+](https://nodejs.org/).

**1. Backend** (num terminal):

```bat
cd backend
go run ./cmd/server
```

A API sobe em http://localhost:8080, com os dados no Supabase (veja
"Banco de dados" abaixo). Sem o Supabase configurado, ela usa o arquivo
`backend/data/boxcubo.json` (criado na primeira vez com as 123 peças da V2).

**2. Frontend** (em outro terminal):

```bat
cd frontend
npm install
npm run dev
```

Abra http://localhost:5173.
Para os testes do backend: `cd backend && go test ./...`

## Banco de dados (Supabase)

Os dados ficam no **Supabase** (Postgres na nuvem), nas tabelas `maquinas`,
`kits`, `pecas`, `pecas_fornecedores`, `receitas`, `legenda`,
`movimentacoes` e `migracoes`. As fotos das peças continuam na pasta
`backend/data/imagens`.

**Configurar (uma vez só):**

1. No Supabase: **SQL Editor › New query**, cole o arquivo
   `backend/supabase/schema.sql` inteiro e clique em **Run** (cria as tabelas).
2. Em **Project Settings › API Keys › Secret keys**, copie a chave secreta
   (`sb_secret_...`).
3. No arquivo `backend/.env` (modelo em `backend/.env.example`), preencha
   `SUPABASE_URL` e `SUPABASE_SECRET_KEY`.
4. Reinicie o backend. Na primeira vez, com o banco vazio, ele **copia os
   dados de `backend/data/boxcubo.json` para o Supabase** e mostra no
   terminal: *"Supabase: banco vazio: dados copiados de data/boxcubo.json"*.
   Das próximas vezes: *"Supabase: dados lidos do Supabase"*.

Como funciona: o backend lê as tabelas ao ligar e, a cada mudança, manda
para o Supabase só as linhas que mudaram. A conversa é pela API REST do
Supabase (HTTPS), com a chave secreta; as tabelas têm RLS ligado e nenhuma
regra, então a chave publicável não acessa nada. A senha do banco não é
usada pelo sistema.

- O `.env` (com a chave) **não vai para o GitHub** (está no `.gitignore`).
- O backend deve ser o único a escrever nas tabelas: editar pelo painel do
  Supabase com o backend ligado pode ser desfeito pela próxima gravação.
  Para editar direto no painel, desligue o backend antes.
- Sem `SUPABASE_SECRET_KEY`, o sistema volta a usar o arquivo
  `backend/data/boxcubo.json` (e avisa no terminal).
- O código está em `backend/internal/store/supabase.go`.

## Formato do SKU

| Item | Formato | Exemplo |
|---|---|---|
| Máquina | `MAQ-<modelo>` | `MAQ-V2` |
| Peça | `<grupo>-<classe>-<final>` | `ELE-RES-1K2` (resistor 1k2 Ω) |

O **final** descreve a peça (valor, medida ou modelo), só com letras e números:
`ELE-RES-10` (10 Ω), `EST-PAR-M4X10` (parafuso M4x10),
`ELE-PES-BALANCA`, `ELE-CTR-RPI3B`. Ao cadastrar uma peça nova, digite o
final na janela; se deixar em branco, o sistema usa um número (`-001`, `-002`…).
Dá para alterar o final depois: o código muda em todo o sistema (lista de
materiais, movimentações e foto) — etiquetas já impressas precisam ser trocadas.

A tabela com os SKUs antigos (numerados, com "BC-") e os novos está em
`backend/internal/store/sku_descritivo.json`. Dados antigos são convertidos
automaticamente quando o backend inicia (as conversões aparecem no terminal
como "Dados atualizados — …" e rodam uma vez só).

## Fornecedores

Cada peça tem uma lista de fornecedores com o **código do item** em cada um;
o primeiro é o **principal**. O fornecedor não é obrigatório: ao salvar uma
peça sem nenhum, aparece um aviso com "Adicionar fornecedor" ou "Salvar sem
fornecedor" (peças fabricadas na BoxCubo não recebem o aviso). Na janela da peça: "+ Adicionar fornecedor",
"Tornar principal" e × para remover. A busca encontra a peça pelo código do
fornecedor.

### Planilhas (Excel)

Todos os botões de baixar (lista de peças, lista de compra, reposição,
peças do fornecedor e pedido) geram um arquivo **.xlsx** já formatado: título
e data no topo, cabeçalho laranja que fica parado ao rolar, colunas na
largura certa, valores em R$ e quantidades com separador de milhar. A
reposição e a lista de compra vêm **separadas por fornecedor**, com subtotal
de cada um e total geral; a lista de peças vem separada pelas seções da tela.
O arquivo é montado no navegador por `frontend/src/planilha.js` (sem
bibliotecas extras).

### Página Fornecedores (barra lateral › Estoque › Fornecedores)

- À esquerda, todos os fornecedores com quantas peças fornecem, quantas estão
  **sem código** do item e quantas estão **para comprar** (abaixo do mínimo).
- Clicando num fornecedor, aparecem as peças dele (de todas as máquinas e
  subconjuntos). O **código no fornecedor** se digita direto na tabela: salva
  ao apertar Enter ou sair do campo (Esc desfaz).
- Abas: *Todas*, *Sem código* (para ir preenchendo) e *Pedido*.
- As colunas **Máquinas** (V2, Eye...) e **Local na máquina** (códigos da
  legenda: PR, CF, ACA...) aparecem nas tabelas e nas planilhas.
- **Pesquisar produto** (logo acima da tabela): procura pelo SKU, nome, código
  do item em **qualquer** fornecedor da peça, **local na máquina** (o código, ex.:
  `CF`, ou o que ele significa, ex.: `cabo de força`) e máquina (ex.: `eye`).
  Várias palavras = todas precisam aparecer (ex.: `parafuso 4x10`).
- **Baixar pedido**: planilha com o que comprar dele para voltar ao estoque mínimo,
  só das peças em que ele é o fornecedor **principal** (peças em que ele é
  alternativo aparecem marcadas como "alternativo" e não entram no pedido).
- **Renomear / juntar**: troca o nome em todas as peças. Se o nome novo já
  existir, os dois viram um só (ex.: juntar "R&A Parafusos" em "1001
  Parafusos"); os códigos preenchidos são mantidos.
- **Não trabalho mais com ele**: passa as peças para outro fornecedor (que
  assume o lugar dele; o código fica em branco, porque era do antigo) ou só
  tira ele das peças. Antes de confirmar, avisa quais peças ficariam sem
  fornecedor.
- **Sem fornecedor** (fim da lista): as peças que não têm nenhum, com um campo
  em cada linha para já escolher o fornecedor e o código.

Grupos: `ELE` Elétrica/Eletrônica · `EST` Estrutural · `EMB` Embalagens/Outros.

Classes: `RES` resistor · `CAP` capacitor · `LED` · `DIO` diodo · `TRS` transistor ·
`CMP` outros componentes · `BPN` barra de pino · `CTR` controladores ·
`CAB` cabeamento · `PCB` · `PES` pesagem · `PER` periféricos (Adicional Correios) ·
`MET` metalúrgico · `ACA` acabamento · `ADE` adesivos · `CAS` cases ·
`PAR` parafusos/ferragens · `EMB` embalagens.

Os itens do **Adicional Correios** são um *kit* opcional da V2: aparecem na
lista marcados e só entram no planejamento quando o kit é marcado.

## Receitas do Controle BoxCubo e subconjuntos

As receitas da **V2, Vision e EYE** e dos subconjuntos **CPU V2** e **Shield
Raspberry** vieram do sistema Controle BoxCubo (aba Materiais, lida em
09/11/2026). Os dados lidos estão em `tools/controle/` e o
`tools/controle/importar.py` monta `backend/internal/store/receitas_controle.json`,
que o backend aplica uma vez (migração `receitas-controle-v1`):

- cada material do Controle virou um SKU — 91 reaproveitaram SKUs que já
  existiam, 43 são novos (ex.: `EST-MET-EYEBRACO1`, `ELE-CTR-NUC`);
- **uma peça = um SKU**, mesmo se usada em várias máquinas (monitor, cabo de
  rede, parafuso M4x10…). A coluna **Máquinas** mostra em quais máquinas a peça vai e o filtro
  **Só compartilhadas** lista só essas peças;
- o estoque passou a ser o do Controle, lançado como **Ajuste de estoque** nas
  Movimentações (pode ser estornado);
- as máquinas que não são mais usadas (Estática V2.5, Dynamic e Slim) saíram
  do sistema; ficaram **V2, Vision e Eye**. As peças delas que também servem
  a outras máquinas continuam cadastradas.

**Subconjuntos** (CPU V2, Shield Raspberry) são peças montadas aqui, com
receita própria (seção *Subconjuntos* da barra lateral):

- **Registrar fabricação**: sai a receita × quantidade e entram as unidades prontas;
- na saída de uma V2 sai **1 CPU pronta** (como no Controle);
- o custo do subconjunto é a soma da receita;
- no **Planejar lote**, se faltar CPU, aparece "fabricar N" e as peças da CPU
  (e do Shield dentro dela) entram na lista de compra, com "para CPU V2";
- o **estoque mínimo** vai para as peças de dentro do subconjunto.

Qualquer peça pode virar subconjunto marcando "Subconjunto fabricado" na
janela da peça.

## Estoque mínimo e reposição

Na página da máquina, a faixa **Estoque mínimo** define para quantas máquinas
vocês querem ter peças sempre em estoque (ex.: 5). O mínimo de cada peça é
calculado pela lista de materiais: quantidade por máquina × número de máquinas.
Peças usadas em mais de um modelo somam os mínimos. Dá para incluir ou não os
itens do kit Adicional Correios.

- Na lista de peças, o estoque fica **laranja** abaixo do mínimo e **vermelho**
  quando zera; embaixo aparece o mínimo da peça. "Só abaixo do mínimo" filtra.
- A barra lateral mostra em **Reposição** quantas peças estão abaixo do mínimo.
- Depois de uma saída, a confirmação lista as peças que ficaram abaixo do mínimo.
- A tela **Reposição** mostra o que comprar para voltar ao mínimo, com custo e
  fornecedor, exporta para o Excel e permite dar entrada quando o material chegar.

## Fotos das peças

Clique numa peça e use **Escolher foto** (ou arraste a imagem para a janela).
Aceita JPG, PNG, WEBP ou GIF até 5 MB. Depois, ao passar o mouse sobre o SKU
em qualquer tabela, aparece um balão com a foto. SKUs com foto têm um ponto
laranja ao lado. As fotos ficam em `backend/data/imagens/` (uma por SKU).

## Estoque

- **Registrar saída** (página da máquina): informe os números de série (um por
  máquina), se leva o kit Adicional Correios e o cliente. A lista de peças é
  descontada do estoque para cada máquina.
  - Se faltar peça, aparece um aviso com o que falta. Em **Continuar mesmo
    assim**, o sistema tira o que existe, deixa a peça em **0** (nunca negativo)
    e registra a falta na movimentação.
  - Um número de série não pode sair duas vezes (a não ser que a saída seja estornada).
- **Registrar saída de peças** (Movimentações): peças avulsas enviadas para
  manutenção ou reposição no cliente. Informe as peças, o cliente e, se quiser,
  o número de série da máquina do cliente (dá para buscar por ele depois).
  Mesmo aviso de falta da saída de máquina.
- **Registrar entrada** (Movimentações ou "Dar entrada desta lista" no
  Planejar lote): soma ao estoque o material que chegou.
- **Movimentações**: histórico de entradas, saídas e estornos, com busca por
  número de série. Clique numa linha para ver peça por peça (antes, depois,
  falta). **Estornar** desfaz uma movimentação devolvendo exatamente o que ela mexeu.

## API

| Método | Rota | O que faz |
|---|---|---|
| GET | `/api/machines` | Máquinas da aba lateral |
| GET | `/api/machines/{id}/bom` | Lista de peças da máquina com custos |
| PUT | `/api/machines/{id}/bom/{sku}` | Adiciona/atualiza peça na máquina |
| DELETE | `/api/machines/{id}/bom/{sku}` | Tira a peça da máquina |
| GET | `/api/machines/{id}/plan?quantidade=43&margem=10&kits=ADDC` | Planejamento de compra |
| GET/POST | `/api/parts` | Lista (`?q=`) / cria peça (gera SKU) |
| GET/PUT/DELETE | `/api/parts/{sku}` | Consulta / edita / exclui peça |
| PUT | `/api/parts/{sku}/sku` | Troca o final do SKU (`{"sufixo": "1K2"}`) |
| GET | `/api/fornecedores` | Nomes de fornecedores já usados |
| GET | `/api/fornecedores/resumo` | Cada fornecedor: peças, sem código, para comprar, valor do pedido |
| GET | `/api/fornecedores/pecas?nome=Fornell` | Peças do fornecedor com código, mínimo e quanto comprar |
| PUT | `/api/fornecedores/codigo` | Grava o código do item (`sku`, `fornecedor`, `codigo`) |
| POST | `/api/fornecedores/renomear` | Renomeia/junta (`de`, `para`) em todas as peças |
| POST | `/api/fornecedores/substituir` | Não trabalha mais com `de`: passa para `para` (ou, vazio, só tira) |
| GET | `/api/fornecedores/sem-fornecedor` | Peças sem nenhum fornecedor |
| GET | `/api/classes`, `/api/kits`, `/api/legenda` | Apoio para os formulários |
| POST | `/api/machines/{id}/saida` | Saída de máquinas (`numeros_serie`, `kits`, `permitir_falta`) — 409 com `faltas` se faltar peça |
| POST | `/api/entradas` | Entrada de material (`itens: [{sku, quantidade}]`) |
| POST | `/api/saidas` | Saída de peças avulsas (`itens`, `cliente`, `numero_serie`, `permitir_falta`) |
| GET | `/api/movements?tipo=&q=` | Histórico de movimentações |
| PUT | `/api/machines/{id}/minimo` | Estoque mínimo da máquina (`maquinas`, `com_kits`) |
| GET | `/api/reposicao` | Peças abaixo do mínimo e sugestão de compra |
| GET | `/api/subconjuntos` | Subconjuntos fabricados |
| POST | `/api/parts/{sku}/fabricar` | Registrar fabricação (`quantidade`, `permitir_falta`) |
| POST/DELETE | `/api/parts/{sku}/imagem` | Envia (multipart, campo `arquivo`) / remove a foto |
| GET | `/api/imagens/{arquivo}` | Serve a foto |
| POST | `/api/movements/{id}/estorno` | Desfaz uma movimentação |

## Pontos a confirmar

- **Estoque:** veio da coluna logo após "onde é usado" na aba Super_Controle.
  Confirmar se é a coluna certa.
- Códigos de "local na máquina" (na planilha: "onde é usado") que não estão na legenda da planilha:
  OS, CL, CR, BL, FB, EMB (marcados com "(?)").
- Itens da BoxCubo Eye (webcam, cabo V8) e "fora da V2" (manípulo,
  parafuso M6x10) **não** foram incluídos na V2.
