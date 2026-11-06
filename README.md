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
| `backend/internal/store/migracoes.go` | Atualizações automáticas de dados antigos |
| `backend/internal/api/api.go` | O "balcão": recebe os pedidos da tela (lista de peças, planejamento, cadastro) |
| `backend/internal/api/estoque.go` | Saídas, entradas e estornos |
| `backend/internal/api/minimo.go` | Estoque mínimo e reposição |
| `backend/internal/api/imagens.go` | Fotos das peças |
| `backend/internal/api/*_test.go` | Testes automáticos (`go test ./...`) |
| `backend/data/` | **Os dados de vocês** (`boxcubo.json`) e as fotos |
| `frontend/src/App.jsx` | A tela inteira: barra lateral + área principal |
| `frontend/src/api.js` | O "garçom": leva os pedidos da tela para o backend |
| `frontend/src/components/` | Cada pedaço da tela (tabela, janelas, barra lateral…) |
| `frontend/src/styles.css` | O visual (cores, fontes) |
| `tools/` | Dados extraídos da planilha e o gerador dos dados iniciais |

## Como rodar (Windows)

Pré-requisitos: [Go 1.22+](https://go.dev/dl/) e [Node.js 18+](https://nodejs.org/).

**1. Backend** (num terminal):

```bat
cd backend
go run ./cmd/server
```

A API sobe em http://localhost:8080. Na primeira execução ela cria
`backend/data/boxcubo.json` já com as 123 peças da V2.

**2. Frontend** (em outro terminal):

```bat
cd frontend
npm install
npm run dev
```

Abra http://localhost:5173.
Para os testes do backend: `cd backend && go test ./...`

## Banco de dados

Por enquanto os dados ficam num arquivo JSON local (`backend/data/boxcubo.json`).
Toda a API usa a interface `store.Store` (`backend/internal/store/store.go`);
quando o banco for escolhido, basta criar uma nova implementação dessa
interface (Postgres, MySQL, SQLite…) e trocar uma linha no
`backend/cmd/server/main.go`. Para voltar aos dados originais da planilha,
apague `backend/data/boxcubo.json` e reinicie o backend.

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
o primeiro é o **principal**. Na janela da peça: "+ Adicionar fornecedor",
"Tornar principal" e × para remover. A busca encontra a peça pelo código do
fornecedor. Nos CSVs (lista de peças, lista de compra e reposição) saem as
colunas *Fornecedor*, *Código no fornecedor* e *Outros fornecedores (código)*;
a lista de compra e a reposição vêm ordenadas por fornecedor principal, para
facilitar os pedidos.

Grupos: `ELE` Elétrica/Eletrônica · `EST` Estrutural · `EMB` Embalagens/Outros.

Classes: `RES` resistor · `CAP` capacitor · `LED` · `DIO` diodo · `TRS` transistor ·
`CMP` outros componentes · `BPN` barra de pino · `CTR` controladores ·
`CAB` cabeamento · `PCB` · `PES` pesagem · `PER` periféricos (Adicional Correios) ·
`MET` metalúrgico · `ACA` acabamento · `ADE` adesivos · `CAS` cases ·
`PAR` parafusos/ferragens · `EMB` embalagens.

Os itens do **Adicional Correios** são um *kit* opcional da V2: aparecem na
lista marcados e só entram no planejamento quando o kit é marcado.

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
  fornecedor, exporta CSV e permite dar entrada quando o material chegar.

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
| GET | `/api/classes`, `/api/kits`, `/api/legenda` | Apoio para os formulários |
| POST | `/api/machines/{id}/saida` | Saída de máquinas (`numeros_serie`, `kits`, `permitir_falta`) — 409 com `faltas` se faltar peça |
| POST | `/api/entradas` | Entrada de material (`itens: [{sku, quantidade}]`) |
| POST | `/api/saidas` | Saída de peças avulsas (`itens`, `cliente`, `numero_serie`, `permitir_falta`) |
| GET | `/api/movements?tipo=&q=` | Histórico de movimentações |
| PUT | `/api/machines/{id}/minimo` | Estoque mínimo da máquina (`maquinas`, `com_kits`) |
| GET | `/api/reposicao` | Peças abaixo do mínimo e sugestão de compra |
| POST/DELETE | `/api/parts/{sku}/imagem` | Envia (multipart, campo `arquivo`) / remove a foto |
| GET | `/api/imagens/{arquivo}` | Serve a foto |
| POST | `/api/movements/{id}/estorno` | Desfaz uma movimentação |

## Pontos a confirmar

- **Estoque:** veio da coluna logo após "onde é usado" na aba Super_Controle.
  Confirmar se é a coluna certa.
- Códigos de "onde é usado" que não estão na legenda da planilha:
  OS, CL, CR, BL, FB, EMB (marcados com "(?)").
- Itens da BoxCubo Eye (webcam, cabo V8) e "fora da V2" (manípulo,
  parafuso M6x10) **não** foram incluídos na V2.
