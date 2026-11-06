// ============================================================
//  FORNECEDORES DA PEÇA (mostrar e editar)
// ============================================================
//  Cada peça pode ter vários fornecedores; cada um com o CÓDIGO DO ITEM
//  naquela loja. O primeiro da lista é o PRINCIPAL (de onde mais compram).

// Mostra os fornecedores da peça com o código do item em cada um.
// Mostra na tabela: nome do fornecedor + código numa caixinha.
export default function Fornecedores({ lista }) {
  if (!lista?.length) return <span className="muted">—</span>
  return (
    <span className="forn-lista">
      {lista.map((f, i) => (
        <span key={i} className={i === 0 ? 'principal' : ''} title={i === 0 ? 'Fornecedor principal' : undefined}>
          {f.nome}
          {f.codigo && <code>{f.codigo}</code>}
        </span>
      ))}
    </span>
  )
}

// Editor da lista de fornecedores (o primeiro é o principal).
// Editor usado na janela da peça: adicionar, remover, "Tornar principal".
// Ao digitar o nome, sugere os fornecedores já cadastrados.
export function FornecedoresEditor({ value, onChange, sugestoes = [] }) {
  const set = (i, campo, v) => onChange(value.map((f, j) => (j === i ? { ...f, [campo]: v } : f)))
  const remover = (i) => onChange(value.filter((_, j) => j !== i))
  const principal = (i) => onChange([value[i], ...value.filter((_, j) => j !== i)])
  return (
    <div className="forn-editor">
      <datalist id="lista-fornecedores">
        {sugestoes.map((n) => <option key={n} value={n} />)}
      </datalist>
      {value.length === 0 && <span className="muted small">Nenhum fornecedor cadastrado.</span>}
      {value.map((f, i) => (
        <div className="forn-linha" key={i}>
          <span className={`forn-tag ${i === 0 ? 'on' : ''}`}>
            {i === 0 ? (
              'Principal'
            ) : (
              <button type="button" className="link" onClick={() => principal(i)} title="Tornar principal">
                Tornar principal
              </button>
            )}
          </span>
          <input
            list="lista-fornecedores"
            value={f.nome}
            onChange={(e) => set(i, 'nome', e.target.value)}
            placeholder="Fornecedor"
          />
          <input
            className="mono"
            value={f.codigo}
            onChange={(e) => set(i, 'codigo', e.target.value)}
            placeholder="Código do item"
          />
          <button type="button" className="icon-btn" onClick={() => remover(i)} title="Remover fornecedor">×</button>
        </div>
      ))}
      <button type="button" className="btn small-btn" onClick={() => onChange([...value, { nome: '', codigo: '' }])}>
        + Adicionar fornecedor
      </button>
    </div>
  )
}
