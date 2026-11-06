// ============================================================
//  JANELA "REGISTRAR SAÍDA DE MÁQUINA" (+ pop-up de falta)
// ============================================================
//  1) digita os números de série (um por máquina);
//  2) marca se leva o kit e o cliente;
//  3) "Enviar": se faltar peça, aparece o pop-up "Não há peças
//     suficientes" (FaltaPopup) com Voltar / Continuar mesmo assim;
//  4) no fim mostra o número da movimentação e as peças que ficaram
//     abaixo do estoque mínimo (AlertaMinimo).

import { useState } from 'react'
import { api, avisarEstoque } from '../api.js'
import { qty } from '../format.js'
import SkuHover from './SkuHover.jsx'

// Separa o texto digitado em números de série (linha, vírgula ou ;).
const separar = (txt) =>
  txt
    .split(/[\n,;]+/)
    .map((s) => s.trim().toUpperCase())
    .filter(Boolean)

// Registrar a saída de máquinas: baixa a lista de peças do estoque.
export default function SaidaModal({ machine, kits, onClose, onDone, onIrParaReposicao }) {
  const [series, setSeries] = useState('')
  const [kitsSel, setKitsSel] = useState([])
  const [cliente, setCliente] = useState('')
  const [obs, setObs] = useState('')
  const [faltas, setFaltas] = useState(null) // pop-up de falta
  const [resultado, setResultado] = useState(null)
  const [erro, setErro] = useState('')
  const [enviando, setEnviando] = useState(false)

  const lista = separar(series)
  const repetidos = lista.filter((s, i) => lista.indexOf(s) !== i)

  // Envia a saída. Primeiro com permitirFalta = false; se o backend avisar
  // falta (código 409), mostra o pop-up. "Continuar mesmo assim" chama de
  // novo com permitirFalta = true.
  async function enviar(permitirFalta) {
    setEnviando(true)
    setErro('')
    try {
      const mov = await api.saida(machine.id, {
        numeros_serie: lista,
        kits: kitsSel,
        cliente,
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

  const toggleKit = (id) => setKitsSel((ks) => (ks.includes(id) ? ks.filter((k) => k !== id) : [...ks, id]))

  if (resultado) {
    const comFalta = resultado.itens.filter((i) => i.falta > 0)
    return (
      <div className="modal-bg">
        <div className="modal">
          <div className="eyebrow">Saída registrada</div>
          <h2>{resultado.id}</h2>
          <p>
            {resultado.quantidade} × {machine.nome} — {resultado.numeros_serie.join(', ')}
            <br />
            <span className="muted">{resultado.itens.length} peças baixadas do estoque.</span>
          </p>
          {comFalta.length > 0 && (
            <div className="alert">
              {comFalta.length} peça(s) saíram com falta e ficaram com estoque 0. A pendência ficou
              registrada nesta movimentação.
            </div>
          )}
          <AlertaMinimo linhas={resultado.abaixo_minimo} />
          <div className="modal-actions">
            {resultado.abaixo_minimo?.length > 0 && onIrParaReposicao && (
              <button className="btn" onClick={onIrParaReposicao}>Ver reposição</button>
            )}
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
        <div className="eyebrow">{machine.nome}</div>
        <h2>Registrar saída de máquina</h2>
        <p className="muted small">
          Cada número de série é uma máquina. A lista de peças da {machine.nome.replace('BoxCubo ', '')} será
          descontada do estoque para cada uma.
        </p>

        <fieldset>
          <legend>Máquinas</legend>
          <label className="full">
            Números de série (um por linha ou separados por vírgula)
            <textarea
              rows={3}
              value={series}
              onChange={(e) => setSeries(e.target.value)}
              placeholder={'BC2-0451\nBC2-0452'}
              autoFocus
            />
            <small className="muted">
              {lista.length} máquina(s)
              {repetidos.length > 0 && <b className="neg"> · repetido: {[...new Set(repetidos)].join(', ')}</b>}
            </small>
          </label>
          {kits.map((k) => (
            <label key={k.id} className="check full" title={k.descricao}>
              <input type="checkbox" checked={kitsSel.includes(k.id)} onChange={() => toggleKit(k.id)} />
              Leva o kit {k.nome}
            </label>
          ))}
        </fieldset>

        <fieldset>
          <legend>Destino</legend>
          <label>
            Cliente / pedido
            <input value={cliente} onChange={(e) => setCliente(e.target.value)} placeholder="opcional" />
          </label>
          <label>
            Observação
            <input value={obs} onChange={(e) => setObs(e.target.value)} placeholder="opcional" />
          </label>
        </fieldset>

        {erro && <div className="alert">{erro}</div>}

        <div className="modal-actions">
          <span style={{ flex: 1 }} />
          <button type="button" className="btn" onClick={onClose}>Cancelar</button>
          <button className="btn primary" disabled={enviando || lista.length === 0 || repetidos.length > 0}>
            {enviando ? 'Enviando…' : 'Enviar'}
          </button>
        </div>
      </form>

      {faltas && (
        <FaltaPopup
          faltas={faltas}
          maquinas={lista.length}
          enviando={enviando}
          onVoltar={() => setFaltas(null)}
          onContinuar={() => enviar(true)}
        />
      )}
    </div>
  )
}

// Lista curta das peças que ficaram abaixo do estoque mínimo depois de uma saída.
// Caixa amarela: peças que ficaram abaixo do mínimo depois da saída
// (mostra até 6 e diz "… e mais N").
export function AlertaMinimo({ linhas }) {
  if (!linhas?.length) return null
  const mostrar = linhas.slice(0, 6)
  return (
    <div className="alerta-minimo">
      <b>
        {linhas.length} peça(s) ficaram abaixo do estoque mínimo
      </b>
      <ul>
        {mostrar.map((l) => (
          <li key={l.sku}>
            <code>{l.sku}</code> {l.nome} — {qty(l.estoque)} de {qty(l.minimo)} {l.unidade}
            {l.zerado && <span className="tag zerado">Zerada</span>}
          </li>
        ))}
        {linhas.length > mostrar.length && <li className="muted">… e mais {linhas.length - mostrar.length}</li>}
      </ul>
    </div>
  )
}

// Pop-up de falta: tabela com o que precisa, o que tem e o que falta.
export function FaltaPopup({ faltas, maquinas, enviando, onVoltar, onContinuar }) {
  return (
    <div className="modal-bg top" onMouseDown={(e) => e.stopPropagation()}>
      <div className="modal popup">
        <div className="eyebrow warn-ink">Atenção</div>
        <h2>Não há peças suficientes</h2>
        <p className="muted small">
          {maquinas ? `Para ${maquinas} máquina(s) faltam` : 'Faltam'} {faltas.length} peça(s) no estoque. Se continuar,
          o sistema tira o que existe, deixa essas peças com estoque <b>0</b> e registra a falta na movimentação.
        </p>
        <div className="table-wrap popup-table">
          <table className="tbl">
            <thead>
              <tr>
                <th>SKU</th>
                <th>Peça</th>
                <th className="r">Precisa</th>
                <th className="r">Estoque</th>
                <th className="r">Falta</th>
              </tr>
            </thead>
            <tbody>
              {faltas.map((f) => (
                <tr key={f.sku}>
                  <td><SkuHover sku={f.sku} nome={f.nome} imagem={f.imagem} /></td>
                  <td>{f.nome}</td>
                  <td className="r">{qty(f.necessario)} {f.unidade}</td>
                  <td className="r">{qty(f.estoque)}</td>
                  <td className="r neg">{qty(f.falta)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="modal-actions">
          <button className="btn" onClick={onVoltar}>Voltar</button>
          <span style={{ flex: 1 }} />
          <button className="btn primary" disabled={enviando} onClick={onContinuar}>
            {enviando ? 'Enviando…' : 'Continuar mesmo assim'}
          </button>
        </div>
      </div>
    </div>
  )
}
