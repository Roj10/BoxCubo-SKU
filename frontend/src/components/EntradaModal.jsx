// ============================================================
//  JANELA "REGISTRAR ENTRADA" / "REGISTRAR SAÍDA DE PEÇAS"
// ============================================================
//  A mesma janela serve para as duas coisas (modo 'entrada' ou 'saida'):
//    - entrada: material que chegou -> soma no estoque;
//    - saída de peças: peças avulsas (manutenção) -> tira do estoque,
//      com cliente e número de série da máquina do cliente.
//  Pode abrir já preenchida (ex.: "Dar entrada desta lista" do
//  Planejar lote ou da Reposição).

import { useEffect, useState } from 'react'
import { api, avisarEstoque } from '../api.js'
import { qty } from '../format.js'
import { FaltaPopup, AlertaMinimo } from './SaidaModal.jsx'

// Registrar entrada (material que chegou) ou saída avulsa de peças
// (manutenção, reposição para cliente). modo = 'entrada' | 'saida'.
// `inicial` pode vir preenchido com a lista de compra do "Planejar lote".
// "linhas" = as peças da lista, cada uma com SKU e quantidade.
export default function EntradaModal({ modo = 'entrada', inicial = [], titulo, onClose, onDone }) {
  const saida = modo === 'saida'
  const [parts, setParts] = useState([])
  const [linhas, setLinhas] = useState(
    inicial.length ? inicial.map((l) => ({ sku: l.sku, quantidade: l.quantidade })) : [{ sku: '', quantidade: '' }],
  )
  const [obs, setObs] = useState(titulo || '')
  const [erro, setErro] = useState('')
  const [enviando, setEnviando] = useState(false)
  const [resultado, setResultado] = useState(null)
  const [cliente, setCliente] = useState('')
  const [serie, setSerie] = useState('')
  const [faltas, setFaltas] = useState(null)

  useEffect(() => {
    api.parts().then(setParts)
  }, [])

  const porSku = Object.fromEntries(parts.map((p) => [p.sku, p]))
  const set = (i, campo, v) => setLinhas((ls) => ls.map((l, j) => (j === i ? { ...l, [campo]: v } : l)))
  const remover = (i) => setLinhas((ls) => ls.filter((_, j) => j !== i))

  // aceita "ELE-RES-10 — 10 Ω" vindo do datalist
  const skuDe = (txt) => txt.split(' — ')[0].trim().toUpperCase()

  const validas = linhas.filter((l) => porSku[skuDe(l.sku)] && Number(l.quantidade) > 0)

  // Envia. Na saída, se faltar peça, aparece o mesmo pop-up de falta da
  // saída de máquina.
  async function enviar(permitirFalta = false) {
    setEnviando(true)
    setErro('')
    const itens = validas.map((l) => ({ sku: skuDe(l.sku), quantidade: Number(l.quantidade) }))
    try {
      const mov = saida
        ? await api.saidaPecas({ itens, cliente, numero_serie: serie, observacao: obs, permitir_falta: permitirFalta })
        : await api.entrada({ itens, observacao: obs })
      setFaltas(null)
      setResultado(mov)
      avisarEstoque()
    } catch (err) {
      if (err.status === 409 && err.data?.faltas) setFaltas(err.data.faltas)
      else setErro(err.message)
    } finally {
      setEnviando(false)
    }
  }

  if (resultado) {
    return (
      <div className="modal-bg">
        <div className="modal">
          <div className="eyebrow">{saida ? 'Saída registrada' : 'Entrada registrada'}</div>
          <h2>{resultado.id}</h2>
          <p className="muted">
            {resultado.itens.length} peça(s) {saida ? 'retiradas do' : 'somadas ao'} estoque.
          </p>
          {resultado.com_falta && (
            <div className="alert">Algumas peças saíram com falta e ficaram com estoque 0. A pendência ficou registrada.</div>
          )}
          <AlertaMinimo linhas={resultado.abaixo_minimo} />
          <div className="modal-actions">
            <span style={{ flex: 1 }} />
            <button className="btn primary" onClick={onDone}>Fechar</button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="modal-bg" onMouseDown={onClose}>
      <form
        className={`modal wide ${saida ? 'modo-saida' : ''}`}
        onMouseDown={(e) => e.stopPropagation()}
        onSubmit={(e) => {
          e.preventDefault()
          enviar(false)
        }}
      >
        <div className="eyebrow">Estoque</div>
        <h2>{saida ? 'Registrar saída de peças' : 'Registrar entrada'}</h2>
        <p className="muted small">
          {saida
            ? 'Peças avulsas enviadas para manutenção ou reposição: as quantidades são retiradas do estoque.'
            : 'Material que chegou: as quantidades são somadas ao estoque de cada peça.'}
        </p>

        <datalist id="lista-pecas">
          {parts.map((p) => <option key={p.sku} value={`${p.sku} — ${p.nome}`} />)}
        </datalist>

        <div className="table-wrap entrada-table">
          <table className="tbl">
            <thead>
              <tr>
                <th>Peça (SKU ou nome)</th>
                <th className="r">Estoque atual</th>
                <th className="r">Quantidade</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {linhas.map((l, i) => {
                const p = porSku[skuDe(l.sku)]
                return (
                  <tr key={i}>
                    <td>
                      <input
                        list="lista-pecas"
                        value={l.sku}
                        onChange={(e) => set(i, 'sku', e.target.value)}
                        placeholder="Digite para buscar…"
                        className="full-input"
                      />
                      {p && <small className="muted">{p.nome}</small>}
                      {l.sku && !p && <small className="neg">SKU não encontrado</small>}
                    </td>
                    <td className={`r ${saida && p && Number(l.quantidade) > p.estoque ? 'neg' : ''}`}>
                      {p ? `${qty(p.estoque)} ${p.unidade}` : '—'}
                    </td>
                    <td className="r">
                      <input
                        type="number"
                        min="0"
                        step="0.01"
                        value={l.quantidade}
                        onChange={(e) => set(i, 'quantidade', e.target.value)}
                        className="qty-input"
                      />
                    </td>
                    <td className="r">
                      <button type="button" className="icon-btn" onClick={() => remover(i)} title="Remover linha">×</button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
        <button type="button" className="btn small-btn" onClick={() => setLinhas([...linhas, { sku: '', quantidade: '' }])}>
          + Adicionar peça
        </button>

        <fieldset>
          <legend>Detalhes</legend>
          {saida && (
            <>
              <label>
                Cliente
                <input value={cliente} onChange={(e) => setCliente(e.target.value)} placeholder="Para quem vão as peças" />
              </label>
              <label>
                Nº de série da máquina do cliente
                <input value={serie} onChange={(e) => setSerie(e.target.value)} placeholder="opcional" />
              </label>
            </>
          )}
          <label className="full">
            {saida ? 'Motivo / observação (manutenção, troca em garantia…)' : 'Observação (nota fiscal, fornecedor…)'}
            <input value={obs} onChange={(e) => setObs(e.target.value)} placeholder="opcional" />
          </label>
        </fieldset>

        {erro && <div className="alert">{erro}</div>}

        <div className="modal-actions">
          <span className="muted small">
            {validas.length} peça(s) prontas para {saida ? 'saída' : 'entrada'}
          </span>
          <span style={{ flex: 1 }} />
          <button type="button" className="btn" onClick={onClose}>Cancelar</button>
          <button className="btn primary" disabled={enviando || validas.length === 0}>
            {enviando ? 'Enviando…' : saida ? 'Registrar saída' : 'Dar entrada'}
          </button>
        </div>
      </form>

      {faltas && (
        <FaltaPopup
          faltas={faltas}
          enviando={enviando}
          onVoltar={() => setFaltas(null)}
          onContinuar={() => enviar(true)}
        />
      )}
    </div>
  )
}
