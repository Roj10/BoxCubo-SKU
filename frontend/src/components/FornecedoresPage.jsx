// ============================================================
//  TELA "FORNECEDORES"
// ============================================================
//  À esquerda: todos os fornecedores, com quantas peças cada um fornece,
//  quantas estão sem o código do item e quanto custa o pedido para repor.
//  À direita: as peças do fornecedor escolhido. Aqui dá para:
//   - digitar o código do item direto na tabela (salva ao sair do campo
//     ou apertar Enter);
//   - filtrar só as peças sem código ou só as que precisam ser compradas;
//   - renomear o fornecedor em todas as peças de uma vez (se o nome novo
//     já existir, os dois viram um só: serve para juntar duplicados);
//   - baixar a lista de peças ou o PEDIDO (o que comprar dele para voltar
//     ao estoque mínimo, só das peças em que ele é o fornecedor principal);
//   - "Não trabalho mais com ele": passar as peças para outro fornecedor ou
//     só tirar ele das peças.
//  No fim da lista fica "Sem fornecedor": as peças que não têm nenhum, com
//  um campo para já colocar o fornecedor ali mesmo.

import { useCallback, useEffect, useMemo, useState } from 'react'
import { api } from '../api.js'
import { money, qty, sigla } from '../format.js'
import { baixarPlanilha, agora } from '../planilha.js'
import SkuHover from './SkuHover.jsx'

// Campo do código do item: guarda o que foi digitado e só salva quando o
// valor muda de verdade. Fica verde rapidinho quando salvou.
function CodigoCampo({ linha, fornecedor, onSalvo }) {
  const [valor, setValor] = useState(linha.codigo)
  const [estado, setEstado] = useState('') // '' | 'salvando' | 'ok' | 'erro'
  useEffect(() => setValor(linha.codigo), [linha.codigo])

  const salvar = async () => {
    if (valor.trim() === linha.codigo) return
    setEstado('salvando')
    try {
      await api.setCodigoFornecedor(linha.sku, fornecedor, valor)
      setEstado('ok')
      onSalvo(linha.sku, valor.trim())
      setTimeout(() => setEstado(''), 1200)
    } catch (e) {
      setEstado('erro')
      alert(e.message)
    }
  }

  return (
    <input
      className={`codigo-inline ${estado} ${valor ? '' : 'vazio'}`}
      value={valor}
      placeholder="sem código"
      onChange={(e) => setValor(e.target.value)}
      onBlur={salvar}
      onKeyDown={(e) => {
        if (e.key === 'Enter') e.target.blur()
        if (e.key === 'Escape') { setValor(linha.codigo); setTimeout(() => e.target.blur()) }
      }}
    />
  )
}

// "Sem fornecedor" é um item especial da lista (não é um fornecedor de verdade).
const SEM = '__sem__'
const buscarPecas = (nome) => (nome === SEM ? api.semFornecedor() : api.fornecedorPecas(nome))

// Linha da lista "Sem fornecedor": escolhe o fornecedor (e o código) e salva
// na peça. A peça some desta lista e passa a aparecer no fornecedor escolhido.
function AdicionarFornecedor({ linha, nomes, onSalvo }) {
  const [nome, setNome] = useState('')
  const [codigo, setCodigo] = useState('')
  const [salvando, setSalvando] = useState(false)
  const salvar = async () => {
    if (!nome.trim()) return
    setSalvando(true)
    try {
      const p = await api.part(linha.sku)
      await api.updatePart(linha.sku, { ...p, fornecedores: [{ nome: nome.trim(), codigo: codigo.trim() }] })
      onSalvo()
    } catch (e) {
      alert(e.message)
      setSalvando(false)
    }
  }
  return (
    <div className="add-forn">
      <input list="lista-forn-pagina" placeholder="Fornecedor" value={nome} onChange={(e) => setNome(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && salvar()} />
      <input className="mono" placeholder="Código do item" value={codigo} onChange={(e) => setCodigo(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && salvar()} />
      <button className="btn small" disabled={!nome.trim() || salvando} onClick={salvar}>Salvar</button>
    </div>
  )
}

// Janela "Não trabalho mais com este fornecedor".
function SubstituirModal({ nome, pecas, todos, onClose, onDone }) {
  const [modo, setModo] = useState('passar') // passar | remover
  const [para, setPara] = useState('')
  const [erro, setErro] = useState('')
  const soEle = pecas.filter((p) => p.outros.length === 0) // ficariam sem fornecedor
  const jaTem = pecas.filter((p) => p.outros.includes(para.trim())).length

  const enviar = async (e) => {
    e.preventDefault()
    try {
      const r = await api.substituirFornecedor(nome, modo === 'passar' ? para : '')
      onDone(modo === 'passar' ? para.trim() : '', r)
    } catch (err) {
      setErro(err.message)
    }
  }

  return (
    <div className="modal-bg" onClick={onClose}>
      <form className="modal" onClick={(e) => e.stopPropagation()} onSubmit={enviar}>
        <div className="eyebrow">Fornecedor</div>
        <h2>Não trabalho mais com {nome}</h2>
        <p className="muted">{nome} está em {pecas.length} {pecas.length === 1 ? 'peça' : 'peças'}. O que fazer com elas?</p>

        <label className="radio">
          <input type="radio" checked={modo === 'passar'} onChange={() => setModo('passar')} />
          <span><b>Passar as peças para outro fornecedor</b> — ele assume o lugar de {nome} (se {nome} era o
            principal, o novo vira principal). O código do item fica em branco para preencher depois.</span>
        </label>
        {modo === 'passar' && (
          <label className="full" style={{ marginLeft: 28 }}>
            Novo fornecedor
            <input list="lista-forn-pagina" value={para} onChange={(e) => setPara(e.target.value)} autoFocus
              placeholder="Escolha um da lista ou digite um novo" />
          </label>
        )}
        {modo === 'passar' && jaTem > 0 && (
          <div className="alert info">
            {jaTem} {jaTem === 1 ? 'peça já tem' : 'peças já têm'} {para.trim()} como fornecedor: nelas, {nome} só sai
            (e o código de {para.trim()} é mantido).
          </div>
        )}

        <label className="radio">
          <input type="radio" checked={modo === 'remover'} onChange={() => setModo('remover')} />
          <span><b>Só tirar {nome} das peças</b> — as peças continuam com os outros fornecedores que já têm.</span>
        </label>
        {modo === 'remover' && soEle.length > 0 && (
          <div className="alert">
            {soEle.length} {soEle.length === 1 ? 'peça só tem' : 'peças só têm'} {nome} e {soEle.length === 1 ? 'vai ficar' : 'vão ficar'} sem
            fornecedor (aparecem em "Sem fornecedor"): {soEle.slice(0, 6).map((p) => p.sku).join(', ')}
            {soEle.length > 6 ? '…' : ''}
          </div>
        )}

        {erro && <div className="alert">{erro}</div>}
        <div className="modal-actions">
          <button type="button" className="btn" onClick={onClose}>Cancelar</button>
          <button className="btn primary" disabled={modo === 'passar' && (!para.trim() || para.trim() === nome)}>
            {modo === 'passar' ? 'Passar as peças' : `Tirar ${nome}`}
          </button>
        </div>
      </form>
    </div>
  )
}

// Janela para renomear / juntar um fornecedor.
function RenomearModal({ nome, todos, onClose, onDone }) {
  const [para, setPara] = useState(nome)
  const [erro, setErro] = useState('')
  const existe = para.trim() !== nome && todos.some((f) => f.nome === para.trim())

  const enviar = async (e) => {
    e.preventDefault()
    try {
      const r = await api.renomearFornecedor(nome, para)
      onDone(r.nome)
    } catch (err) {
      setErro(err.message)
    }
  }

  return (
    <div className="modal-bg" onClick={onClose}>
      <form className="modal" onClick={(e) => e.stopPropagation()} onSubmit={enviar}>
        <div className="eyebrow">Fornecedor</div>
        <h2>Renomear / juntar</h2>
        <p className="muted">
          Troca <b>{nome}</b> pelo nome novo em todas as peças. Se o nome novo já existir, os dois viram um
          fornecedor só (os códigos preenchidos são mantidos).
        </p>
        <label className="full" style={{ marginTop: 14 }}>
          Novo nome
          <input list="lista-fornecedores" value={para} onChange={(e) => setPara(e.target.value)} autoFocus />
          <datalist id="lista-fornecedores">
            {todos.filter((f) => f.nome !== nome).map((f) => <option key={f.nome} value={f.nome} />)}
          </datalist>
        </label>
        {existe && (
          <div className="alert info">
            <b>{para.trim()}</b> já existe: as peças de <b>{nome}</b> vão passar para ele.
          </div>
        )}
        {erro && <div className="alert">{erro}</div>}
        <div className="modal-actions">
          <button type="button" className="btn" onClick={onClose}>Cancelar</button>
          <button className="btn primary" disabled={!para.trim() || para.trim() === nome}>
            {existe ? 'Juntar' : 'Renomear'}
          </button>
        </div>
      </form>
    </div>
  )
}

export default function FornecedoresPage() {
  const [resumo, setResumo] = useState(null)
  const [busca, setBusca] = useState('')
  const [atual, setAtual] = useState('') // fornecedor aberto
  const [pecas, setPecas] = useState(null)
  const [filtro, setFiltro] = useState('todas') // todas | sem-codigo | comprar
  const [q, setQ] = useState('') // pesquisa de produto
  const [legenda, setLegenda] = useState({}) // "onde é usado": código -> o que é (ACA -> Acabamento...)
  useEffect(() => {
    api.legenda().then(setLegenda).catch(() => {})
  }, [])
  const [renomear, setRenomear] = useState(false)
  const [substituir, setSubstituir] = useState(false)
  const [semCount, setSemCount] = useState(0) // quantas peças estão sem fornecedor
  const [aviso, setAviso] = useState('')
  const [erro, setErro] = useState('')

  const carregarResumo = useCallback(
    () =>
      api.fornecedoresResumo().then((r) => {
        setResumo(r)
        api.semFornecedor().then((s) => setSemCount(s.length)).catch(() => {})
        return r
      }).catch((e) => setErro(e.message)),
    [],
  )

  // ao abrir: busca o resumo e já abre o primeiro fornecedor
  useEffect(() => {
    carregarResumo().then((r) => r?.length && setAtual((a) => a || r[0].nome))
    const recarregar = () => carregarResumo()
    window.addEventListener('estoque-mudou', recarregar)
    return () => window.removeEventListener('estoque-mudou', recarregar)
  }, [carregarResumo])

  // sempre que troca o fornecedor, busca as peças dele
  // nenhum aberto (ex.: depois de tirar um fornecedor): abre o primeiro
  useEffect(() => {
    if (!atual && resumo?.length) setAtual(resumo[0].nome)
  }, [atual, resumo])

  useEffect(() => {
    if (!atual) return
    setPecas(null)
    setQ('')
    buscarPecas(atual).then(setPecas).catch((e) => setErro(e.message))
  }, [atual])

  const listaFiltrada = useMemo(() => {
    const t = busca.trim().toLowerCase()
    return (resumo || []).filter((f) => !t || f.nome.toLowerCase().includes(t))
  }, [resumo, busca])

  // Pesquisa: procura o texto no SKU, no nome da peça, no código do item
  // (em qualquer fornecedor da peça), no "onde é usado" (o código, ex.: CAB,
  // e o que ele significa) e nas máquinas em que a peça é usada.
  // Várias palavras = todas precisam aparecer (ex.: "parafuso 4x10").
  const pesquisados = useMemo(() => {
    if (!pecas) return null
    const palavras = q.trim().toLowerCase().split(/\s+/).filter(Boolean)
    if (!palavras.length) return pecas
    return pecas.filter((p) => {
      const onde = (p.onde_usado || []).map((o) => `${o} ${legenda[o] || ''}`).join(' ')
      const txt = [p.sku, p.nome, p.codigo, ...(p.codigos || []), onde, ...(p.usada_em || []).map((m) => `${m} ${sigla(m)}`)]
        .join(' ')
        .toLowerCase()
      return palavras.every((w) => txt.includes(w))
    })
  }, [pecas, q, legenda])

  const linhas = useMemo(() => {
    if (!pesquisados) return []
    if (filtro === 'sem-codigo') return pesquisados.filter((p) => !p.codigo)
    if (filtro === 'comprar') return pesquisados.filter((p) => p.principal && p.comprar > 0)
    return pesquisados
  }, [pesquisados, filtro])

  // códigos "onde é usado" (passando o mouse mostra o que significam)
  const ondeChips = (p) =>
    (p.onde_usado || []).map((o) => (
      <span key={o} className="chip" title={legenda[o] || 'Código fora da legenda'}>{o}</span>
    ))

  // campo de pesquisa, logo acima da tabela
  const campoPesquisa = (
    <div className="forn-pesquisa">
      <input
        className="search"
        placeholder="Pesquisar: SKU, nome, código do fornecedor ou onde é usado"
        value={q}
        onChange={(e) => setQ(e.target.value)}
      />
      {q && (
        <>
          <span className="muted">{linhas.length} de {pecas?.length ?? 0}</span>
          <button className="link" onClick={() => setQ('')}>limpar</button>
        </>
      )}
    </div>
  )

  const info = resumo?.find((f) => f.nome === atual)
  const nomes = (resumo || []).map((f) => f.nome)

  // depois de colocar fornecedor numa peça da lista "Sem fornecedor"
  const recarregarSem = () => {
    buscarPecas(SEM).then(setPecas)
    carregarResumo()
  }
  const pedido = (pecas || []).filter((p) => p.principal && p.comprar > 0)

  // quando um código é salvo: atualiza a linha e o contador "sem código"
  const codigoSalvo = (sku, codigo) => {
    setPecas((ps) => ps.map((p) => (p.sku === sku ? { ...p, codigo } : p)))
    carregarResumo()
  }

  const nomeArquivo = (s) => s.normalize('NFD').replace(/[\u0300-\u036f]/g, '').replace(/[^\w]+/g, '-').replace(/^-|-$/g, '').toLowerCase()

  // Planilha com todas as peças deste fornecedor (a pesquisa e a aba
  // escolhida valem: baixa o que está na tela).
  const exportarPecas = () =>
    baixarPlanilha(`fornecedor-${nomeArquivo(atual)}.xlsx`, {
      titulo: `Peças do fornecedor ${atual}`,
      subtitulo: `${linhas.length} peças · gerado em ${agora()}`,
      aba: atual,
      colunas: [
        { titulo: 'Código no fornecedor' },
        { titulo: 'SKU' },
        { titulo: 'Peça' },
        { titulo: 'Un.' },
        { titulo: 'Principal?' },
        { titulo: 'Estoque', tipo: 'num' },
        { titulo: 'Mínimo', tipo: 'num' },
        { titulo: 'Comprar', tipo: 'num', destaque: true },
        { titulo: 'Valor unit.', tipo: 'preco' },
        { titulo: 'Usada em' },
        { titulo: 'Onde é usado' },
        { titulo: 'Outros fornecedores' },
      ],
      linhas: linhas.map((p) => [
        p.codigo, p.sku, p.nome, p.unidade, p.principal ? 'Sim' : 'Não (alternativo)', p.estoque, p.minimo || null,
        p.comprar || null, p.valor_unitario, p.usada_em.map(sigla).join(', '), (p.onde_usado || []).join(', '),
        p.outros.join(', '),
      ]),
    })

  // Pedido para mandar ao fornecedor: o código dele primeiro, quantidade e total.
  const exportarPedido = () =>
    baixarPlanilha(`pedido-${nomeArquivo(atual)}.xlsx`, {
      titulo: `Pedido de compra — ${atual}`,
      subtitulo: `${pedido.length} ${pedido.length === 1 ? 'item' : 'itens'} para voltar ao estoque mínimo · gerado em ${agora()}`,
      aba: 'Pedido',
      colunas: [
        { titulo: 'Código no fornecedor' },
        { titulo: 'Descrição' },
        { titulo: 'Un.' },
        { titulo: 'Quantidade', tipo: 'num', destaque: true },
        { titulo: 'Valor unit.', tipo: 'preco' },
        { titulo: 'Total', tipo: 'dinheiro', total: true },
        { titulo: 'SKU BoxCubo' },
      ],
      linhas: pedido.map((p) => [p.codigo, p.nome, p.unidade, p.comprar, p.valor_unitario, p.custo, p.sku]),
    })

  return (
    <div>
      <header className="page-head">
        <div>
          <div className="eyebrow">Compras</div>
          <h1>Fornecedores</h1>
          <span className="muted">Quem fornece cada peça, o código do item em cada um e o pedido para repor.</span>
        </div>
      </header>

      {erro && <div className="alert">{erro}</div>}

      <div className="forn-layout">
        {/* ---- lista de fornecedores ---- */}
        <aside className="forn-lista">
          <input
            className="search"
            placeholder={`Buscar entre ${resumo?.length ?? '…'} fornecedores`}
            value={busca}
            onChange={(e) => setBusca(e.target.value)}
          />
          <div className="forn-itens">
            {listaFiltrada.map((f) => (
              <button
                key={f.nome}
                className={`forn-item ${f.nome === atual ? 'active' : ''}`}
                onClick={() => { setAtual(f.nome); setFiltro('todas') }}
              >
                <span className="forn-nome">{f.nome}</span>
                <span className="forn-meta">
                  {f.pecas} {f.pecas === 1 ? 'peça' : 'peças'}
                  {f.sem_codigo > 0 && <em className="sem-cod" title="Peças sem o código do item">{f.sem_codigo} sem código</em>}
                  {f.abaixo_minimo > 0 && <em className="abaixo" title="Peças para comprar (abaixo do mínimo)">{f.abaixo_minimo} p/ comprar</em>}
                </span>
              </button>
            ))}
            {resumo && listaFiltrada.length === 0 && <p className="muted pad">Nenhum fornecedor com esse nome.</p>}
            <button className={`forn-item sem ${atual === SEM ? 'active' : ''}`} onClick={() => setAtual(SEM)}>
              <span className="forn-nome">Sem fornecedor</span>
              <span className="forn-meta">
                {semCount} {semCount === 1 ? 'peça' : 'peças'}
                {semCount > 0 && <em className="abaixo">para definir</em>}
              </span>
            </button>
          </div>
          <datalist id="lista-forn-pagina">
            {nomes.map((n) => <option key={n} value={n} />)}
          </datalist>
        </aside>

        {/* ---- peças do fornecedor escolhido ---- */}
        <section className="forn-detalhe">
          {aviso && <div className="alert info" style={{ marginTop: 0, marginBottom: 14 }}>{aviso}</div>}
          {atual === SEM && (
            <>
              <div className="forn-titulo"><h2>Sem fornecedor</h2></div>
              <p className="muted">
                Peças que não têm nenhum fornecedor (as fabricadas na BoxCubo não entram aqui). Escolha o fornecedor
                na linha e clique em Salvar; a peça passa a aparecer nele.
              </p>
              {pecas === null ? <p className="muted">Carregando…</p> : pecas.length === 0 ? (
                <div className="empty small-empty ok-empty"><h2>Todas as peças têm fornecedor</h2></div>
              ) : (
                <>
                {campoPesquisa}
                <div className="table-wrap">
                  <table className="tbl">
                    <thead>
                      <tr><th>SKU</th><th>Peça</th><th>Fornecedor e código</th><th className="r">Estoque</th><th>Usada em</th><th>Onde é usado</th></tr>
                    </thead>
                    <tbody>
                      {linhas.map((p) => (
                        <tr key={p.sku}>
                          <td><SkuHover sku={p.sku} nome={p.nome} imagem={p.imagem} /></td>
                          <td>{p.nome}</td>
                          <td><AdicionarFornecedor linha={p} nomes={nomes} onSalvo={recarregarSem} /></td>
                          <td className="r">{qty(p.estoque)} {p.unidade}</td>
                          <td>{p.usada_em.map((m) => <span key={m} className="chip maq" title={m}>{sigla(m)}</span>)}</td>
                          <td>{ondeChips(p)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                {linhas.length === 0 && <p className="muted pad">Nenhuma peça encontrada para "{q}".</p>}
                </>
              )}
            </>
          )}
          {info && (
            <>
              <div className="forn-titulo">
                <h2>{info.nome}</h2>
                <button className="btn small" onClick={() => setRenomear(true)}>Renomear / juntar</button>
                <button className="btn small danger" onClick={() => setSubstituir(true)} disabled={!pecas}>
                  Não trabalho mais com ele
                </button>
              </div>

              <section className="cards">
                <div className="card">
                  <span title="Em quantas ele é o fornecedor principal">Peças · principal em {info.principal}</span>
                  <strong>{info.pecas}</strong>
                </div>
                <div className={`card ${info.sem_codigo ? 'warn' : ''}`}>
                  <span>Sem código do item</span>
                  <strong>{info.sem_codigo}</strong>
                </div>
                <div className={`card ${info.abaixo_minimo ? 'warn' : ''}`}>
                  <span>Para comprar</span>
                  <strong>{info.abaixo_minimo}</strong>
                </div>
                <div className="card accent">
                  <span>Valor do pedido</span>
                  <strong>{money(info.custo_repor)}</strong>
                </div>
              </section>

              <div className="toolbar">
                <div className="tabs">
                  <button className={filtro === 'todas' ? 'on' : ''} onClick={() => setFiltro('todas')}>
                    Todas ({pecas?.length ?? '…'})
                  </button>
                  <button className={filtro === 'sem-codigo' ? 'on' : ''} onClick={() => setFiltro('sem-codigo')}>
                    Sem código ({info.sem_codigo})
                  </button>
                  <button className={filtro === 'comprar' ? 'on' : ''} onClick={() => setFiltro('comprar')}>
                    Pedido ({info.abaixo_minimo})
                  </button>
                </div>
                <button className="btn" onClick={exportarPecas} disabled={!pecas}>Baixar peças</button>
                <button className="btn primary no-auto" onClick={exportarPedido} disabled={!pedido.length}
                  title={pedido.length ? '' : 'Nada abaixo do mínimo para este fornecedor'}>
                  Baixar pedido
                </button>
              </div>

              {pecas !== null && pecas.length > 0 && campoPesquisa}
              {pecas === null ? (
                <p className="muted">Carregando…</p>
              ) : q && linhas.length === 0 ? (
                <p className="muted pad">Nenhuma peça encontrada para "{q}".</p>
              ) : linhas.length === 0 ? (
                <div className="empty small-empty ok-empty">
                  <h2>{filtro === 'sem-codigo' ? 'Todas as peças têm código' : filtro === 'comprar' ? 'Nada para comprar' : 'Nenhuma peça'}</h2>
                  {filtro === 'comprar' && <p>As peças deste fornecedor estão acima do estoque mínimo.</p>}
                </div>
              ) : (
                <div className="table-wrap">
                  <table className="tbl">
                    <thead>
                      <tr>
                        <th>SKU</th>
                        <th>Peça</th>
                        <th>Código no fornecedor</th>
                        <th className="r">Estoque</th>
                        <th className="r">Mínimo</th>
                        <th className="r">Comprar</th>
                        <th className="r">Valor unit.</th>
                        <th>Usada em</th>
                        <th>Onde é usado</th>
                      </tr>
                    </thead>
                    <tbody>
                      {linhas.map((p) => (
                        <tr key={p.sku}>
                          <td><SkuHover sku={p.sku} nome={p.nome} imagem={p.imagem} /></td>
                          <td>
                            {p.nome}
                            {!p.principal && (
                              <span className="tag" title={`Fornecedor principal: ${p.outros[0] || '—'}`}>alternativo</span>
                            )}
                          </td>
                          <td><CodigoCampo linha={p} fornecedor={atual} onSalvo={codigoSalvo} /></td>
                          <td className={`r ${p.minimo > 0 && p.estoque < p.minimo ? (p.estoque <= 0 ? 'est-zero' : 'est-baixo') : ''}`}>
                            {qty(p.estoque)} {p.unidade}
                          </td>
                          <td className="r">{p.minimo ? qty(p.minimo) : '—'}</td>
                          <td className={`r ${p.comprar > 0 ? 'neg' : ''}`}>
                            {p.comprar > 0 ? qty(p.comprar) : '—'}
                          </td>
                          <td className="r">{money(p.valor_unitario)}</td>
                          <td>
                            {p.usada_em.map((m) => (
                              <span key={m} className={`chip maq ${p.usada_em.length > 1 ? 'compart' : ''}`} title={m}>
                                {sigla(m)}
                              </span>
                            ))}
                          </td>
                          <td>{ondeChips(p)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </section>
      </div>

      {substituir && pecas && (
        <SubstituirModal
          nome={atual}
          pecas={pecas}
          todos={resumo || []}
          onClose={() => setSubstituir(false)}
          onDone={async (novo, r) => {
            setSubstituir(false)
            const antigo = atual
            await carregarResumo()
            const sem = r.sem_fornecedor.length
            setAviso(
              (novo ? `As ${r.pecas} peças de ${antigo} passaram para ${novo}.` : `${antigo} foi tirado de ${r.pecas} peças.`) +
                (sem ? ` ${sem} ${sem === 1 ? 'ficou' : 'ficaram'} sem fornecedor.` : ''),
            )
            setTimeout(() => setAviso(''), 8000)
            setAtual(novo || (sem ? SEM : ''))
          }}
        />
      )}

      {renomear && (
        <RenomearModal
          nome={atual}
          todos={resumo || []}
          onClose={() => setRenomear(false)}
          onDone={async (novo) => {
            setRenomear(false)
            await carregarResumo()
            setAtual(novo)
            setPecas(null)
            api.fornecedorPecas(novo).then(setPecas)
          }}
        />
      )}
    </div>
  )
}
