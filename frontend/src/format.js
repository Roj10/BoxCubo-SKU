// ============================================================
//  FORMATAÇÃO: deixa números, dinheiro e arquivos no padrão brasileiro
// ============================================================

const brl = new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' })
const num = new Intl.NumberFormat('pt-BR', { maximumFractionDigits: 3 })

// Dinheiro: 7504.53 -> "R$ 7.504,53" (sem valor -> "—").
export const money = (v) => (v === null || v === undefined ? '—' : brl.format(v))
// Quantidade: 1352 -> "1.352"; 0.25 -> "0,25".
export const qty = (v) => (v === null || v === undefined ? '—' : num.format(v))

// ---- fornecedores ----
// texto curto para busca
export const fornTexto = (fs = []) => (fs || []).map((f) => `${f.nome} ${f.codigo}`).join(' ')

// ordena a lista de compra por fornecedor principal (facilita fazer os pedidos)
// (peças sem fornecedor vão para o fim)
// Ordena a lista de compra por fornecedor principal (peças sem fornecedor
// vão para o fim), para montar um pedido por fornecedor.
export const porFornecedor = (a, b) => {
  const fa = a.fornecedores?.[0]?.nome || ''
  const fb = b.fornecedores?.[0]?.nome || ''
  if (!fa !== !fb) return fa ? -1 : 1
  return fa.localeCompare(fb, 'pt-BR')
}

// Sigla curta de cada máquina (coluna "Usada em"): "Estática V2" -> "V2".
const SIGLA = { 'Estática V2': 'V2', Vision: 'VIS', Eye: 'EYE' }
export const sigla = (nome) => SIGLA[nome] || nome

// Nome do grupo de uma linha na planilha: o fornecedor principal.
export const grupoFornecedor = (fs) => fs?.[0]?.nome || 'Sem fornecedor'
// Os fornecedores além do principal, com código: "Mercado Livre (ML123); Kabum".
export const outros = (fs) =>
  (fs || []).slice(1).map((f) => (f.codigo ? `${f.nome} (${f.codigo})` : f.nome)).join('; ')
