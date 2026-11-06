// ============================================================
//  JANELA "ESTOQUE MÍNIMO" da máquina
// ============================================================
//  Pergunta "manter peças para quantas máquinas?" e se os itens dos kits
//  entram no mínimo. 0 = sem estoque mínimo.

import { useState } from 'react'
import { api, avisarEstoque } from '../api.js'

// Configura o estoque mínimo da máquina em "peças para N máquinas".
export default function MinimoModal({ machine, temKits, onClose, onSaved }) {
  const [n, setN] = useState(machine.minimo_maquinas || 5)
  const [comKits, setComKits] = useState(!!machine.minimo_com_kits)
  const [erro, setErro] = useState('')
  const [salvando, setSalvando] = useState(false)

  async function salvar(e) {
    e.preventDefault()
    setSalvando(true)
    try {
      const m = await api.setMinimo(machine.id, Math.max(0, Number(n) || 0), comKits)
      avisarEstoque()
      onSaved(m)
    } catch (err) {
      setErro(err.message)
      setSalvando(false)
    }
  }

  const nome = machine.nome.replace('BoxCubo ', '')

  return (
    <div className="modal-bg" onMouseDown={onClose}>
      <form className="modal" onMouseDown={(e) => e.stopPropagation()} onSubmit={salvar}>
        <div className="eyebrow">{machine.nome}</div>
        <h2>Estoque mínimo</h2>
        <p className="muted small">
          Quantas máquinas {nome} vocês querem conseguir montar a qualquer momento? O sistema calcula o mínimo de cada
          peça pela lista de materiais: por exemplo, com 5 máquinas o mínimo da balança é 5 e o dos sensores
          ultrassônicos é 50. Se uma peça é usada em mais de um modelo, os mínimos se somam.
        </p>

        <fieldset>
          <legend>Manter peças para</legend>
          <label>
            Número de máquinas
            <input type="number" min="0" step="1" value={n} onChange={(e) => setN(e.target.value)} autoFocus />
            <small className="muted">0 = sem estoque mínimo para esta máquina</small>
          </label>
          {temKits && (
            <label className="check">
              <input type="checkbox" checked={comKits} onChange={(e) => setComKits(e.target.checked)} />
              Incluir os itens dos kits (Adicional Correios)
            </label>
          )}
        </fieldset>

        {erro && <div className="alert">{erro}</div>}

        <div className="modal-actions">
          <span style={{ flex: 1 }} />
          <button type="button" className="btn" onClick={onClose}>Cancelar</button>
          <button className="btn primary" disabled={salvando}>{salvando ? 'Salvando…' : 'Salvar'}</button>
        </div>
      </form>
    </div>
  )
}
