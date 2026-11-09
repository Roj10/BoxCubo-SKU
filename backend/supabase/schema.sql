-- ============================================================
--  TABELAS DO BOXCUBO SKU NO SUPABASE
-- ============================================================
--  Como usar (uma vez só):
--   1. Abra o projeto no supabase.com > SQL Editor > New query.
--   2. Cole TODO este arquivo e clique em "Run".
--  Pode rodar de novo sem problema: só cria o que ainda não existe.
--
--  Segurança: todas as tabelas ficam com RLS ligado e SEM regras de
--  acesso. Ou seja, a chave publicável (sb_publishable_...) não lê nem
--  grava nada. Só o backend, com a chave SECRETA, acessa os dados.

-- Máquinas (V2, Vision, Eye...) e o estoque mínimo de cada uma
create table if not exists public.maquinas (
  id               text primary key,          -- v2, vision, eye
  sku              text not null default '',   -- MAQ-V2
  nome             text not null default '',
  apelido          text not null default '',
  tecnologia       text not null default '',
  ativa            boolean not null default false,
  minimo_maquinas  integer not null default 0, -- estoque mínimo: peças para N máquinas
  minimo_com_kits  boolean not null default false,
  ordem            integer not null default 0  -- ordem na barra lateral
);

-- Kits opcionais (ex.: ADDC - Adicional Correios)
create table if not exists public.kits (
  id         text primary key,
  nome       text not null default '',
  descricao  text not null default '',
  ordem      integer not null default 0
);

-- Peças (uma linha por SKU)
create table if not exists public.pecas (
  sku             text primary key,              -- ELE-RES-10
  nome            text not null default '',
  grupo           text not null default '',
  classe          text not null default '',
  unidade         text not null default 'und',
  tipo            text not null default 'peca',  -- peca, consumivel, servico
  valor_unitario  double precision,              -- vazio = sem preço
  estoque         double precision not null default 0,
  imagem          text not null default '',      -- endereço da foto
  fabricado       boolean not null default false, -- subconjunto feito na BoxCubo
  ordem           integer not null default 0
);

-- Fornecedores de cada peça (ordem 0 = fornecedor principal)
create table if not exists public.pecas_fornecedores (
  sku     text not null,
  ordem   integer not null,
  nome    text not null default '',
  codigo  text not null default '',  -- código do item no fornecedor
  primary key (sku, ordem)
);
create index if not exists pecas_fornecedores_nome on public.pecas_fornecedores (nome);

-- Receitas (lista de materiais) de cada máquina e de cada subconjunto
create table if not exists public.receitas (
  dono        text not null,             -- v2, eye... ou ELE-SUB-CPUV2
  ordem       integer not null,
  sku         text not null,
  quantidade  double precision not null default 0,  -- por máquina
  onde_usado  jsonb not null default '[]',          -- ["PR","CF"]
  kit         text not null default '',             -- '' = item padrão
  primary key (dono, ordem)
);
create index if not exists receitas_sku on public.receitas (sku);

-- Legenda do "onde é usado" (PR = ..., CF = Cabo de força...)
create table if not exists public.legenda (
  codigo     text primary key,
  descricao  text not null default ''
);

-- Movimentações de estoque (saídas, entradas, estornos, fabricação...)
create table if not exists public.movimentacoes (
  id             text primary key,     -- MOV-000001
  data           text not null default '',
  tipo           text not null default '',
  machine_id     text not null default '',
  quantidade     integer not null default 0,
  numeros_serie  jsonb not null default '[]',
  kits           jsonb not null default '[]',
  cliente        text not null default '',
  observacao     text not null default '',
  com_falta      boolean not null default false,
  estornado      boolean not null default false,
  estornado_por  text not null default '',
  estorno_de     text not null default '',
  itens          jsonb not null default '[]'  -- peças movimentadas
);

-- Atualizações de dados já aplicadas (controle interno do sistema)
create table if not exists public.migracoes (
  nome   text primary key,
  ordem  integer not null default 0
);

-- RLS ligado, sem regras: só a chave secreta (backend) acessa.
alter table public.maquinas            enable row level security;
alter table public.kits                enable row level security;
alter table public.pecas               enable row level security;
alter table public.pecas_fornecedores  enable row level security;
alter table public.receitas            enable row level security;
alter table public.legenda             enable row level security;
alter table public.movimentacoes       enable row level security;
alter table public.migracoes           enable row level security;

-- O backend usa a chave secreta (papel service_role): garante o acesso.
grant select, insert, update, delete on
  public.maquinas, public.kits, public.pecas, public.pecas_fornecedores,
  public.receitas, public.legenda, public.movimentacoes, public.migracoes
  to service_role;

-- Faz a API do Supabase enxergar as tabelas novas na hora
notify pgrst, 'reload schema';
