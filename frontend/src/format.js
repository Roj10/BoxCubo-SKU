// ============================================================
//  FORMATAÇÃO: deixa números, dinheiro e arquivos no padrão brasileiro
// ============================================================

const brl = new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' })
const num = new Intl.NumberFormat('pt-BR', { maximumFractionDigits: 3 })

// Dinheiro: 7504.53 -> "R$ 7.504,53" (sem valor -> "—").
export const money = (v) => (v === null || v === undefined ? '—' : brl.format(v))
// Quantidade: 1352 -> "1.352"; 0.25 -> "0,25".
export const qty = (v) => (v === null || v === undefined ? '—' : num.format(v))

// Gera o arquivo .csv (abre no Excel) e baixa no computador.
// Usa ";" para separar colunas e vírgula nos decimais, que é o padrão do
// Excel em português. O "\ufeff" no começo faz o Excel entender acentos.
export function downloadCSV(filename, rows) {
  const esc = (v) => {
    // números com vírgula decimal, para o Excel em português
    const s = v === null || v === undefined ? '' : typeof v === 'number' ? String(v).replace('.', ',') : String(v)
    return /[";\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
  }
  // ; como separador para abrir direto no Excel em português
  const csv = '﻿' + rows.map((r) => r.map(esc).join(';')).join('\n')
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }))
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

// ---- fornecedores ----
// [{nome, codigo}] -> colunas para CSV: principal, código no principal, demais
// Colunas de fornecedor usadas em todos os CSVs.
export const FORN_COLS = ['Fornecedor', 'Código no fornecedor', 'Outros fornecedores (código)']
// Transforma a lista de fornecedores nas 3 colunas do CSV:
// fornecedor principal | código no principal | outros (com código).
export function fornCols(fs = []) {
  const [p, ...resto] = fs || []
  return [
    p?.nome || '',
    p?.codigo || '',
    resto.map((f) => (f.codigo ? `${f.nome} (${f.codigo})` : f.nome)).join('; '),
  ]
}
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
