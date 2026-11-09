// ============================================================
//  JANELA "NOVA PEÇA" / "EDITAR PEÇA"
// ============================================================
//  Partes da janela:
//    - Foto da peça (escolher, arrastar, trocar, remover);
//    - Peça: final do SKU, nome, tipo, unidade, preço, estoque e
//      fornecedores com código;
//    - Na máquina: quantidade por máquina, kit e onde é usada.
//  Ao salvar, manda tudo para o backend na ordem certa (troca de SKU
//  primeiro, depois os dados, depois a lista da máquina, depois a foto).

import { useEffect, useState } from 'react'
import { api } from '../api.js'
import CodigosInput from './CodigosInput.jsx'
import { FornecedoresEditor } from './Fornecedores.jsx'

const UNIDADES = ['und', 'pct', 'rolo', 'm', 'serviço'] // pct = pacote (ex.: pacote com 2 ou 10)
const TIPOS = [
  ['peca', 'Peça'],
  ['consumivel', 'Consumível'],
  ['servico', 'Serviço'],
]

// Formulário para criar uma peça nova ou editar uma peça da lista da máquina.
// "novo" = true quando é "+ Nova peça"; false quando é edição.
// "f" guarda tudo que está digitado no formulário.
export default function PartForm({ machine, line, classes, kits, legenda, onClose, onSaved }) {
  const novo = !line
  const p = line?.peca || {}
  const [f, setF] = useState({
    prefixo: classes[0]?.prefixo || '',
    sufixo: novo ? '' : line.sku.slice(line.sku.lastIndexOf('-') + 1),
    nome: p.nome || '',
    unidade: p.unidade || 'und',
    tipo: p.tipo || 'peca',
    fornecedores: (p.fornecedores || []).map((x) => ({ nome: x.nome, codigo: x.codigo || '' })),
    valor_unitario: p.valor_unitario ?? '',
    estoque: p.estoque ?? 0,
    fabricado: !!p.fabricado,
    quantidade: line?.quantidade ?? 1,
    onde_usado: line?.onde_usado || [],
    kit: line?.kit || '',
  })
  const [erro, setErro] = useState('')
  const [salvando, setSalvando] = useState(false)
  // aviso "salvar sem fornecedor?" (não é obrigatório, só um lembrete)
  const [avisoSemForn, setAvisoSemForn] = useState(false)
  const [nomesForn, setNomesForn] = useState([])
  useEffect(() => {
    api.fornecedores().then(setNomesForn).catch(() => {})
  }, [])
  const [arquivo, setArquivo] = useState(null) // foto nova escolhida
  const [preview, setPreview] = useState(p.imagem || '')
  const [removerFoto, setRemoverFoto] = useState(false)

  useEffect(() => {
    if (!arquivo) return
    const url = URL.createObjectURL(arquivo)
    setPreview(url)
    return () => URL.revokeObjectURL(url)
  }, [arquivo])

  // Confere a foto escolhida (precisa ser imagem e ter até 5 MB).
  function escolherFoto(file) {
    if (!file) return
    if (!file.type.startsWith('image/')) return setErro('Escolha um arquivo de imagem (JPG, PNG, WEBP).')
    if (file.size > 5 * 1024 * 1024) return setErro('A imagem passa de 5 MB.')
    setErro('')
    setArquivo(file)
    setRemoverFoto(false)
  }

  const set = (k) => (e) => setF({ ...f, [k]: e.target.value })
  const classeSel = classes.find((c) => c.prefixo === f.prefixo)
  const prefixoAtual = novo ? f.prefixo : line.sku.slice(0, line.sku.lastIndexOf('-'))
  const sufixoOriginal = novo ? '' : line.sku.slice(line.sku.lastIndexOf('-') + 1)
  const sufixoMudou = !novo && f.sufixo !== sufixoOriginal
  // só letras e números, em maiúsculas (o hífen separa as partes do SKU)
  const setSufixo = (e) => setF({ ...f, sufixo: e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, '').slice(0, 16) })

  // Botão "Salvar". Se a peça ficou sem fornecedor, primeiro mostra um
  // aviso (dá para salvar mesmo assim). Peças fabricadas na BoxCubo não
  // precisam de fornecedor, então não avisa.
  async function salvar(e, mesmoSemFornecedor = false) {
    e?.preventDefault()
    const temFornecedor = f.fornecedores.some((x) => x.nome.trim())
    if (!temFornecedor && !f.fabricado && !mesmoSemFornecedor) {
      setAvisoSemForn(true)
      return
    }
    setAvisoSemForn(false)
    setSalvando(true)
    setErro('')
    const part = {
      nome: f.nome,
      grupo: novo ? classeSel?.grupo : p.grupo,
      classe: novo ? classeSel?.classe : p.classe,
      unidade: f.unidade,
      tipo: f.tipo,
      fornecedores: f.fornecedores,
      valor_unitario: f.valor_unitario === '' ? null : Number(f.valor_unitario),
      estoque: Number(f.estoque) || 0,
      fabricado: f.fabricado,
    }
    const bom = {
      quantidade: Number(f.quantidade) || 0,
      onde_usado: f.onde_usado,
      kit: f.kit,
    }
    try {
      let sku = line?.sku
      if (novo) {
        const criada = await api.createPart({
          ...part,
          prefixo: f.prefixo,
          sufixo: f.sufixo,
          machine_id: machine.id,
          ...bom,
        })
        sku = criada.sku
      } else {
        if (sufixoMudou) sku = (await api.renameSku(line.sku, f.sufixo)).sku
        await api.updatePart(sku, part)
        await api.upsertBOM(machine.id, sku, bom)
      }
      if (arquivo) await api.uploadImagem(sku, arquivo)
      else if (removerFoto && p.imagem) await api.removerImagem(sku)
      onSaved()
    } catch (err) {
      setErro(err.message)
      setSalvando(false)
    }
  }

  // Botão "Tirar da máquina": tira da lista desta máquina, mas a peça
  // continua cadastrada (pode estar em outras máquinas).
  async function remover() {
    if (!window.confirm(`Tirar ${line.sku} da lista da ${machine.nome}? A peça continua cadastrada.`)) return
    try {
      await api.deleteBOM(machine.id, line.sku)
      onSaved()
    } catch (err) {
      setErro(err.message)
    }
  }

  return (
    <div className="modal-bg" onMouseDown={onClose}>
      <form className="modal" onMouseDown={(e) => e.stopPropagation()} onSubmit={salvar}>
        <h2>{novo ? 'Nova peça' : 'Editar peça'}</h2>
        {!novo && <code className="sku-big">{line.sku}</code>}

        <div
          className="foto-box"
          onDragOver={(e) => e.preventDefault()}
          onDrop={(e) => {
            e.preventDefault()
            escolherFoto(e.dataTransfer.files[0])
          }}
        >
          <div className="foto-prev">
            {preview && !removerFoto ? <img src={preview} alt="Foto da peça" /> : <span className="muted">Sem foto</span>}
          </div>
          <div className="foto-acoes">
            <b>Foto da peça</b>
            <span className="muted small">
              Aparece ao passar o mouse no SKU. Arraste uma imagem aqui ou escolha um arquivo (JPG, PNG ou WEBP, até 5 MB).
            </span>
            <div className="foto-botoes">
              <label className="btn small-file">
                {preview && !removerFoto ? 'Trocar foto' : 'Escolher foto'}
                <input type="file" accept="image/*" hidden onChange={(e) => escolherFoto(e.target.files[0])} />
              </label>
              {preview && !removerFoto && (
                <button
                  type="button"
                  className="btn danger"
                  onClick={() => {
                    setArquivo(null)
                    setRemoverFoto(true)
                  }}
                >
                  Remover
                </button>
              )}
            </div>
          </div>
        </div>

        <fieldset>
          <legend>Peça</legend>
          {novo && (
            <label className="full">
              Classe (define o SKU)
              <select value={f.prefixo} onChange={set('prefixo')}>
                {classes.map((c) => (
                  <option key={c.prefixo} value={c.prefixo}>
                    {c.prefixo} — {c.grupo} › {c.classe}
                  </option>
                ))}
              </select>
            </label>
          )}
          <label className="full">
            Final do SKU
            <span className="sku-final">
              <code>{prefixoAtual}-</code>
              <input
                value={f.sufixo}
                onChange={setSufixo}
                placeholder={novo ? 'ex.: 47, 1K2, M4X10' : ''}
                required={!novo}
              />
            </span>
            <small className={sufixoMudou ? 'warn-ink' : 'muted'}>
              {novo
                ? f.sufixo
                  ? `Vai ficar: ${f.prefixo}-${f.sufixo}`
                  : `Use algo que descreva a peça (valor, medida, modelo). Em branco = número automático (${f.prefixo}-001…).`
                : sufixoMudou
                  ? `O código vai mudar para ${prefixoAtual}-${f.sufixo || '?'} em todo o sistema (lista, movimentações e foto). Etiquetas já impressas precisam ser trocadas.`
                  : 'Letras e números, sem espaços. Dá para alterar se o código não estiver claro.'}
            </small>
          </label>
          <label className="full">
            Nome
            <input required value={f.nome} onChange={set('nome')} autoFocus />
          </label>
          <label>
            Tipo
            <select value={f.tipo} onChange={set('tipo')}>
              {TIPOS.map(([v, t]) => <option key={v} value={v}>{t}</option>)}
            </select>
          </label>
          <label>
            Unidade
            <select value={f.unidade} onChange={set('unidade')}>
              {UNIDADES.map((u) => <option key={u}>{u}</option>)}
            </select>
          </label>
          <label>
            Valor unitário (R$)
            <input type="number" step="0.001" min="0" value={f.valor_unitario} onChange={set('valor_unitario')} />
          </label>
          <label>
            Estoque
            <input type="number" step="0.01" min="0" value={f.estoque} onChange={set('estoque')} />
          </label>
          <label className="check full" title="Peça montada aqui, com receita própria (ex.: CPU, Shield Raspberry)">
            <input
              type="checkbox"
              checked={f.fabricado}
              onChange={(e) => setF({ ...f, fabricado: e.target.checked })}
            />
            Subconjunto fabricado na BoxCubo (tem receita própria e aparece em "Subconjuntos")
          </label>
          <div className="full campo">
            <span className="campo-titulo">Fornecedores e código do item</span>
            <FornecedoresEditor
              value={f.fornecedores}
              onChange={(v) => setF({ ...f, fornecedores: v })}
              sugestoes={nomesForn}
            />
          </div>
        </fieldset>

        <fieldset>
          <legend>Na {machine.nome}</legend>
          <label>
            Quantidade por máquina
            <input type="number" step="0.001" min="0" value={f.quantidade} onChange={set('quantidade')} />
          </label>
          <label>
            Kit
            <select value={f.kit} onChange={set('kit')}>
              <option value="">Item padrão</option>
              {kits.map((k) => <option key={k.id} value={k.id}>{k.nome}</option>)}
            </select>
          </label>
          <div className="full campo">
            <span className="campo-titulo">Onde é usado</span>
            <CodigosInput
              value={f.onde_usado}
              onChange={(v) => setF({ ...f, onde_usado: v })}
              legenda={legenda}
            />
          </div>
        </fieldset>

        {erro && <div className="alert">{erro}</div>}

        {avisoSemForn && (
          <div className="alerta-minimo aviso-forn">
            <b>Esta peça está sem fornecedor.</b> Não é obrigatório, mas sem ele a peça não entra no pedido de
            nenhum fornecedor e fica na lista "Sem fornecedor".
            <div className="aviso-acoes">
              <button
                type="button"
                className="btn"
                onClick={() => {
                  setAvisoSemForn(false)
                  setF({ ...f, fornecedores: [...f.fornecedores.filter((x) => x.nome.trim()), { nome: '', codigo: '' }] })
                }}
              >
                Adicionar fornecedor
              </button>
              <button type="button" className="btn primary" onClick={() => salvar(null, true)}>
                Salvar sem fornecedor
              </button>
            </div>
          </div>
        )}

        <div className="modal-actions">
          {!novo && (
            <button type="button" className="btn danger" onClick={remover}>Tirar da máquina</button>
          )}
          <span style={{ flex: 1 }} />
          <button type="button" className="btn" onClick={onClose}>Cancelar</button>
          <button className="btn primary" disabled={salvando}>{salvando ? 'Salvando…' : 'Salvar'}</button>
        </div>
      </form>
    </div>
  )
}
