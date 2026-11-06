// ============================================================
//  API (lado da tela): o "garçom" que leva pedidos ao backend
// ============================================================
//  Toda conversa com o backend passa por aqui. Cada função é um pedido
//  pronto, ex.: api.bom('v2') = "me traz a lista de peças da V2".
//  Assim, se um endereço mudar, só este arquivo precisa ser ajustado.
//
//  Em desenvolvimento, o Vite (vite.config.js) repassa tudo que começa com
//  /api para o backend em localhost:8080.

// Cliente da API Go. Em desenvolvimento o Vite repassa /api para localhost:8080.
// Função base usada por todos os pedidos: envia, espera a resposta e, se
// deu erro, transforma num aviso legível (a mensagem que vem do backend).
async function req(method, url, body) {
  const res = await fetch(url, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })
  if (res.status === 204) return null
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const err = new Error(data.erro || `Erro ${res.status}`)
    err.status = res.status
    err.data = data // ex.: { faltas: [...] } quando falta peça na saída
    throw err
  }
  return data
}

// Avisa as telas (ex.: contador da barra lateral) que o estoque mudou.
// "Grito" interno: quando o estoque muda (saída, entrada, edição...), avisa
// as partes da tela que dependem dele (ex.: o número laranja da Reposição)
// para se atualizarem sozinhas.
export const avisarEstoque = () => window.dispatchEvent(new Event('estoque-mudou'))

// O "cardápio" de pedidos que a tela pode fazer ao backend.
export const api = {
  machines: () => req('GET', '/api/machines'),
  bom: (id) => req('GET', `/api/machines/${id}/bom`),
  plan: (id, { quantidade, margem, kits }) =>
    req('GET', `/api/machines/${id}/plan?quantidade=${quantidade}&margem=${margem}&kits=${kits.join(',')}`),
  kits: () => req('GET', '/api/kits'),
  legenda: () => req('GET', '/api/legenda'),
  classes: () => req('GET', '/api/classes'),
  fornecedores: () => req('GET', '/api/fornecedores'),
  parts: (q = '') => req('GET', `/api/parts?q=${encodeURIComponent(q)}`),
  createPart: (body) => req('POST', '/api/parts', body),
  updatePart: (sku, part) => req('PUT', `/api/parts/${encodeURIComponent(sku)}`, part),
  renameSku: (sku, sufixo) => req('PUT', `/api/parts/${encodeURIComponent(sku)}/sku`, { sufixo }),
  deletePart: (sku) => req('DELETE', `/api/parts/${encodeURIComponent(sku)}`),
  upsertBOM: (id, sku, item) => req('PUT', `/api/machines/${id}/bom/${encodeURIComponent(sku)}`, item),
  deleteBOM: (id, sku) => req('DELETE', `/api/machines/${id}/bom/${encodeURIComponent(sku)}`),

  // fotos das peças
  uploadImagem: async (sku, arquivo) => {
    const fd = new FormData()
    fd.append('arquivo', arquivo)
    const res = await fetch(`/api/parts/${encodeURIComponent(sku)}/imagem`, { method: 'POST', body: fd })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.erro || `Erro ${res.status}`)
    return data
  },
  removerImagem: (sku) => req('DELETE', `/api/parts/${encodeURIComponent(sku)}/imagem`),

  // estoque
  saida: (id, body) => req('POST', `/api/machines/${id}/saida`, body),
  entrada: (body) => req('POST', '/api/entradas', body),
  saidaPecas: (body) => req('POST', '/api/saidas', body),
  movements: ({ machine = '', tipo = '', q = '' } = {}) =>
    req('GET', `/api/movements?machine=${machine}&tipo=${tipo}&q=${encodeURIComponent(q)}`),
  estorno: (id, observacao) => req('POST', `/api/movements/${id}/estorno`, { observacao }),

  // estoque mínimo
  setMinimo: (id, maquinas, comKits) => req('PUT', `/api/machines/${id}/minimo`, { maquinas, com_kits: comKits }),
  reposicao: () => req('GET', '/api/reposicao'),
}
