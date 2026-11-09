// ============================================================
//  JANELA "REGISTRAR FABRICAÇÃO" de um subconjunto (ex.: CPU)
// ============================================================
//  Informa quantas unidades foram montadas. O sistema tira do estoque a
//  receita x quantidade e soma as unidades prontas ao estoque do
//  subconjunto. Se faltar peça, mostra o mesmo pop-up de falta das saídas.

import { useState } from 'react'
import { api, avisarEstoque } from '../api.js'
import { FaltaPopup, AlertaMinimo } from './SaidaModal.jsx'

export default function FabricarModal({ sub, onClose, onDone }) {
  const [quantidade, setQuantidade] = useState(1)
  const [obs, setObs] = useState('')
  const [faltas, setFaltas] = useState(null)
  const [resultado, setResultado] = useState(null)
  const [erro, setErro] = useState('')
  const [enviando, setEnviando] = useState(false)

  // Envia; se o backend avisar falta (409), mostra o pop-up.
  // "Continuar mesmo assim" envia de novo com permitir_falta = true.
  async function enviar(permitirFalta) {
    setEnviando(true)
    setErro('')
    try {
      const mov = await api.fabricar(sub.sku, {
        quantidade: Number(quantidade),
        observacao: obs,
        permitir_falta: permitirFalta,
      })
      setFaltas(null)
      setResultado(mov)
      avisarEstoque()
    } catch (e) {
      if (e.status === 409 && e.data?.faltas) setFaltas(e.data.faltas)
      else setErro(e.message)
    } finally {
      setEnviando(false)
    }
  }

  if (resultado) {
    return (
      <div className="modal-bg">
        <div className="modal">
          <div className="eyebrow">Fabricação registrada</div>
          <h2>{resultado.id}</h2>
          <p>
            {resultado.quantidade} × {sub.nome} entraram no estoque.
            <br />
            <span className="muted">{resultado.itens.length - 1} peças da receita baixadas do estoque.</span>
          </p>
          {resultado.com_falta && (
            <div className="alert">
              Algumas peças faltaram e ficaram com estoque 0. A pendência ficou registrada nesta movimentação.
            </div>
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
        className="modal"
        onMouseDown={(e) => e.stopPropagation()}
        onSubmit={(e) => {
          e.preventDefault()
          enviar(false)
        }}
      >
        <div className="eyebrow">{sub.nome}</div>
        <h2>Registrar fabricação</h2>
        <p className="muted small">
          Sai do estoque a receita de cada unidade montada, e as unidades prontas entram no estoque do subconjunto.
        </p>
        <fieldset>
          <legend>Fabricação</legend>
          <label>
            Quantidade montada
            <input
              type="number"
              min="1"
              step="1"
              value={quantidade}
              onChange={(e) => setQuantidade(e.target.value)}
              autoFocus
            />
          </label>
          <label>
            Observação
            <input value={obs} onChange={(e) => setObs(e.target.value)} placeholder="opcional (ex.: lote, responsável)" />
          </label>
        </fieldset>
        {erro && <div className="alert">{erro}</div>}
        <div className="modal-actions">
          <span style={{ flex: 1 }} />
          <button type="button" className="btn" onClick={onClose}>Cancelar</button>
          <button className="btn primary" disabled={enviando || !(Number(quantidade) >= 1)}>
            {enviando ? 'Enviando…' : 'Registrar'}
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
