// ============================================================
//  PLANILHA EXCEL (.xlsx) — as listas que o sistema baixa
// ============================================================
//  Antes as listas saíam em CSV (texto separado por ";"), que abre no
//  Excel todo "cru": colunas estreitas, sem cabeçalho destacado, valores
//  sem R$. Aqui o sistema monta um arquivo do Excel de verdade, já
//  arrumado:
//   - título e data no topo;
//   - cabeçalho laranja, que fica parado ao rolar a planilha, com filtro;
//   - colunas na largura do conteúdo;
//   - dinheiro em "R$ 1.234,56" e quantidades com separador de milhar;
//   - (opcional) linhas agrupadas, por exemplo por fornecedor, com o
//     subtotal de cada grupo e o total geral no fim.
//
//  Como funciona por dentro: um arquivo .xlsx é uma pasta compactada (zip)
//  com alguns arquivos de texto (XML) que descrevem a planilha. Este
//  arquivo escreve esses textos e monta o zip, sem precisar de nenhuma
//  biblioteca extra.
//
//  Como usar:
//    baixarPlanilha('reposicao.xlsx', {
//      titulo: 'Reposição de estoque',
//      subtitulo: '35 peças abaixo do mínimo',
//      colunas: [
//        { titulo: 'SKU' },                          // texto
//        { titulo: 'Comprar', tipo: 'num', destaque: true },
//        { titulo: 'Valor unit.', tipo: 'preco' },     // R$ com até 4 casas
//        { titulo: 'Custo', tipo: 'dinheiro', total: true },
//      ],
//      linhas: [{ grupo: 'Fornell', valores: ['ELE-RES-10', 20, 3.5] }, ...],
//    })
//  Cada linha pode ser só a lista de valores, ou { grupo, valores }.
//  As colunas com "total: true" ganham subtotal por grupo e total geral.

// ---------- estilos (fonte, cor, formato de número) ----------
// O número de cada estilo é a posição dele na lista "cellXfs" abaixo.
const E = {
  normal: 0,
  titulo: 1,
  subtitulo: 2,
  cabecalho: 3,
  num: 4,
  dinheiro: 5,
  grupo: 6,
  subtotalTexto: 7,
  subtotalDinheiro: 8,
  totalTexto: 9,
  totalDinheiro: 10,
  numDestaque: 11,
  subtotalNum: 12,
  totalNum: 13,
  preco: 14, // valor unitário: mostra até 4 casas (ex.: R$ 0,035 de um parafuso)
  cabecalhoDireita: 15,
}

const ESTILOS = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<numFmts count="4">
<numFmt numFmtId="164" formatCode="&quot;R$&quot;\\ #,##0.00\\ \\ "/>
<numFmt numFmtId="165" formatCode="#,##0.###\\ \\ ;-#,##0.###\\ \\ ;0\\ \\ "/>
<numFmt numFmtId="166" formatCode="&quot;R$&quot;\\ #,##0.00##\\ \\ "/>
<numFmt numFmtId="167" formatCode="\\ @"/>
</numFmts>
<fonts count="6">
<font><sz val="11"/><name val="Calibri"/></font>
<font><b/><sz val="15"/><color rgb="FF121212"/><name val="Calibri"/></font>
<font><i/><sz val="10"/><color rgb="FF7A7A7A"/><name val="Calibri"/></font>
<font><b/><sz val="11"/><color rgb="FFFFFFFF"/><name val="Calibri"/></font>
<font><b/><sz val="11"/><name val="Calibri"/></font>
<font><b/><sz val="11"/><color rgb="FFC05A00"/><name val="Calibri"/></font>
</fonts>
<fills count="5">
<fill><patternFill patternType="none"/></fill>
<fill><patternFill patternType="gray125"/></fill>
<fill><patternFill patternType="solid"><fgColor rgb="FFDA8301"/><bgColor indexed="64"/></patternFill></fill>
<fill><patternFill patternType="solid"><fgColor rgb="FFF3EFE9"/><bgColor indexed="64"/></patternFill></fill>
<fill><patternFill patternType="solid"><fgColor rgb="FFFCE9CC"/><bgColor indexed="64"/></patternFill></fill>
</fills>
<borders count="3">
<border><left/><right/><top/><bottom/><diagonal/></border>
<border><left/><right/><top/><bottom style="thin"><color rgb="FFE2DDD5"/></bottom><diagonal/></border>
<border><left/><right/><top style="medium"><color rgb="FF121212"/></top><bottom/><diagonal/></border>
</borders>
<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
<cellXfs count="16">
<xf numFmtId="167" fontId="0" fillId="0" borderId="1" xfId="0" applyNumberFormat="1" applyBorder="1"><alignment vertical="top" indent="1"/></xf>
<xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/>
<xf numFmtId="0" fontId="2" fillId="0" borderId="0" xfId="0" applyFont="1"/>
<xf numFmtId="167" fontId="3" fillId="2" borderId="0" xfId="0" applyNumberFormat="1" applyFont="1" applyFill="1" applyAlignment="1"><alignment vertical="center" wrapText="1" indent="1"/></xf>
<xf numFmtId="165" fontId="0" fillId="0" borderId="1" xfId="0" applyNumberFormat="1" applyBorder="1" applyAlignment="1"><alignment horizontal="right" vertical="top" indent="1"/></xf>
<xf numFmtId="164" fontId="0" fillId="0" borderId="1" xfId="0" applyNumberFormat="1" applyBorder="1" applyAlignment="1"><alignment horizontal="right" vertical="top" indent="1"/></xf>
<xf numFmtId="0" fontId="4" fillId="3" borderId="0" xfId="0" applyFont="1" applyFill="1" applyAlignment="1"><alignment indent="1"/></xf>
<xf numFmtId="0" fontId="4" fillId="0" borderId="0" xfId="0" applyFont="1" applyAlignment="1"><alignment horizontal="right"/></xf>
<xf numFmtId="164" fontId="4" fillId="0" borderId="0" xfId="0" applyNumberFormat="1" applyFont="1" applyAlignment="1"><alignment horizontal="right" indent="1"/></xf>
<xf numFmtId="0" fontId="4" fillId="4" borderId="2" xfId="0" applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1"><alignment horizontal="right"/></xf>
<xf numFmtId="164" fontId="4" fillId="4" borderId="2" xfId="0" applyNumberFormat="1" applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1"><alignment horizontal="right" indent="1"/></xf>
<xf numFmtId="165" fontId="5" fillId="0" borderId="1" xfId="0" applyNumberFormat="1" applyFont="1" applyBorder="1" applyAlignment="1"><alignment horizontal="right" vertical="top" indent="1"/></xf>
<xf numFmtId="165" fontId="4" fillId="0" borderId="0" xfId="0" applyNumberFormat="1" applyFont="1" applyAlignment="1"><alignment horizontal="right" indent="1"/></xf>
<xf numFmtId="165" fontId="4" fillId="4" borderId="2" xfId="0" applyNumberFormat="1" applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1"><alignment horizontal="right" indent="1"/></xf>
<xf numFmtId="166" fontId="0" fillId="0" borderId="1" xfId="0" applyNumberFormat="1" applyBorder="1" applyAlignment="1"><alignment horizontal="right" vertical="top" indent="1"/></xf>
<xf numFmtId="0" fontId="3" fillId="2" borderId="0" xfId="0" applyFont="1" applyFill="1" applyAlignment="1"><alignment horizontal="right" vertical="center" wrapText="1" indent="1"/></xf>
</cellXfs>
<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>
</styleSheet>`

// ---------- pedacinhos de XML ----------
// Troca caracteres especiais (& < > ") para não quebrar o arquivo.
const esc = (s) =>
  String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    // caracteres de controle não são aceitos pelo Excel
    .replace(/[\u0000-\u0008\u000B\u000C\u000E-\u001F]/g, '')

// Número da coluna -> letra do Excel: 0 -> A, 25 -> Z, 26 -> AA...
function letra(n) {
  let s = ''
  n += 1
  while (n > 0) {
    const r = (n - 1) % 26
    s = String.fromCharCode(65 + r) + s
    n = Math.floor((n - 1) / 26)
  }
  return s
}

// Uma célula. Números viram números de verdade (o Excel soma e filtra).
function celula(ref, valor, estilo) {
  if (valor === null || valor === undefined || valor === '') {
    return estilo !== undefined ? `<c r="${ref}" s="${estilo}"/>` : ''
  }
  if (typeof valor === 'number' && Number.isFinite(valor)) {
    return `<c r="${ref}" s="${estilo}"><v>${valor}</v></c>`
  }
  return `<c r="${ref}" s="${estilo}" t="inlineStr"><is><t xml:space="preserve">${esc(valor)}</t></is></c>`
}

// Estilo de cada tipo de coluna.
function estiloDe(col) {
  if (col.tipo === 'dinheiro') return E.dinheiro
  if (col.tipo === 'preco') return E.preco
  if (col.tipo === 'num') return col.destaque ? E.numDestaque : E.num
  return E.normal
}

// Largura da coluna (em "caracteres"), pelo maior conteúdo, entre 8 e 55.
function largura(col, i, linhas) {
  if (col.largura) return col.largura
  let max = String(col.titulo).length * 0.9
  for (const l of linhas) {
    const v = l.valores[i]
    if (v === null || v === undefined) continue
    const n = typeof v === 'number' ? (col.tipo === 'dinheiro' || col.tipo === 'preco' ? 15 : String(v).length + 4) : String(v).length + 2
    if (n > max) max = n
  }
  return Math.min(60, Math.max(8, Math.ceil(max * 1.1 + 2)))
}

// Monta o XML da aba (a planilha em si).
function montarAba({ titulo, subtitulo, colunas, linhas }) {
  const L = linhas.map((l) => (Array.isArray(l) ? { valores: l } : l))
  const nCol = colunas.length
  const ultima = letra(nCol - 1)
  const comTotal = colunas.map((c, i) => (c.total ? i : -1)).filter((i) => i >= 0)
  const temGrupo = L.some((l) => l.grupo !== undefined)
  // a etiqueta "Subtotal"/"Total" fica na coluna antes do primeiro total
  const colEtiqueta = comTotal.length ? Math.max(0, comTotal[0] - 1) : 0

  const xml = []
  let r = 0
  const linha = (cels, extra = '') => {
    r += 1
    xml.push(`<row r="${r}"${extra}>${cels.join('')}</row>`)
    return r
  }

  linha([celula(`A${r + 1}`, titulo, E.titulo)], ' ht="22" customHeight="1"')
  linha([celula(`A${r + 1}`, subtitulo, E.subtitulo)])
  linha([])
  const linhaCab = linha(
    colunas.map((c, i) => celula(`${letra(i)}${r + 1}`, c.titulo, c.tipo ? E.cabecalhoDireita : E.cabecalho)),
    ' ht="30" customHeight="1"',
  )
  const inicioDados = r + 1

  // soma das colunas com total (por grupo e geral)
  const zera = () => Object.fromEntries(comTotal.map((i) => [i, 0]))
  let somaGrupo = zera()
  const somaGeral = zera()
  let grupoAtual

  const linhaSoma = (rotulo, somas, estTexto, estDinheiro, estNum) =>
    linha(
      colunas.map((c, i) => {
        const ref = `${letra(i)}${r + 1}`
        if (i === colEtiqueta && !comTotal.includes(i)) return celula(ref, rotulo, estTexto)
        if (comTotal.includes(i)) return celula(ref, Math.round(somas[i] * 1000) / 1000, c.tipo === 'dinheiro' ? estDinheiro : estNum)
        return estTexto === E.totalTexto ? celula(ref, null, estTexto) : ''
      }),
    )
  const fechaGrupo = () => {
    if (grupoAtual !== undefined && comTotal.length) {
      linhaSoma(`Subtotal ${grupoAtual}`, somaGrupo, E.subtotalTexto, E.subtotalDinheiro, E.subtotalNum)
    }
  }

  for (const l of L) {
    if (temGrupo && l.grupo !== grupoAtual) {
      fechaGrupo()
      grupoAtual = l.grupo
      somaGrupo = zera()
      const qtd = L.filter((x) => x.grupo === l.grupo).length
      linha(
        colunas.map((_, i) =>
          celula(`${letra(i)}${r + 1}`, i === 0 ? `${l.grupo}  (${qtd} ${qtd === 1 ? 'item' : 'itens'})` : null, E.grupo),
        ),
      )
    }
    linha(colunas.map((c, i) => celula(`${letra(i)}${r + 1}`, l.valores[i], estiloDe(c))))
    for (const i of comTotal) {
      const v = Number(l.valores[i]) || 0
      somaGrupo[i] += v
      somaGeral[i] += v
    }
  }
  if (temGrupo) fechaGrupo()
  const fimDados = r
  if (comTotal.length && L.length) {
    linha([])
    linhaSoma('Total geral', somaGeral, E.totalTexto, E.totalDinheiro, E.totalNum)
  }

  const larguras = colunas
    .map((c, i) => `<col min="${i + 1}" max="${i + 1}" width="${largura(c, i, L)}" customWidth="1"/>`)
    .join('')

  // cabeçalho parado ao rolar (linhas acima de "inicioDados" ficam fixas)
  const painel = `<sheetViews><sheetView workbookViewId="0" showGridLines="0"><pane ySplit="${linhaCab}" topLeftCell="A${inicioDados}" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>`
  // filtro no cabeçalho (só sem grupos: com grupos, o filtro bagunçaria os subtotais)
  const filtro = !temGrupo && L.length ? `<autoFilter ref="A${linhaCab}:${ultima}${fimDados}"/>` : ''

  return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheetPr><pageSetUpPr fitToPage="1"/></sheetPr>${painel}<sheetFormatPr defaultRowHeight="16"/><cols>${larguras}</cols><sheetData>${xml.join('')}</sheetData>${filtro}
<pageMargins left="0.4" right="0.4" top="0.5" bottom="0.5" header="0.3" footer="0.3"/>
<pageSetup orientation="landscape" fitToWidth="1" fitToHeight="0"/>
</worksheet>`
}

// ---------- o "zip" (o .xlsx é um zip) ----------
// Código de conferência (CRC32) que todo arquivo dentro de um zip precisa.
const TABELA_CRC = (() => {
  const t = new Uint32Array(256)
  for (let n = 0; n < 256; n++) {
    let c = n
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
    t[n] = c >>> 0
  }
  return t
})()
function crc32(bytes) {
  let c = 0xffffffff
  for (let i = 0; i < bytes.length; i++) c = TABELA_CRC[(c ^ bytes[i]) & 0xff] ^ (c >>> 8)
  return (c ^ 0xffffffff) >>> 0
}

// Junta os arquivos num zip simples (sem compressão — o Excel aceita).
function zip(arquivos) {
  const enc = new TextEncoder()
  const partes = []
  const central = []
  let pos = 0
  for (const [nome, texto] of arquivos) {
    const nomeB = enc.encode(nome)
    const dados = enc.encode(texto)
    const crc = crc32(dados)
    const loc = new DataView(new ArrayBuffer(30))
    loc.setUint32(0, 0x04034b50, true)
    loc.setUint16(4, 20, true)
    loc.setUint16(6, 0x0800, true) // nomes em UTF-8
    loc.setUint16(8, 0, true) // sem compressão
    loc.setUint16(10, 0, true)
    loc.setUint16(12, 0x21, true)
    loc.setUint32(14, crc, true)
    loc.setUint32(18, dados.length, true)
    loc.setUint32(22, dados.length, true)
    loc.setUint16(26, nomeB.length, true)
    loc.setUint16(28, 0, true)
    partes.push(new Uint8Array(loc.buffer), nomeB, dados)

    const cen = new DataView(new ArrayBuffer(46))
    cen.setUint32(0, 0x02014b50, true)
    cen.setUint16(4, 20, true)
    cen.setUint16(6, 20, true)
    cen.setUint16(8, 0x0800, true)
    cen.setUint16(10, 0, true)
    cen.setUint16(12, 0, true)
    cen.setUint16(14, 0x21, true)
    cen.setUint32(16, crc, true)
    cen.setUint32(20, dados.length, true)
    cen.setUint32(24, dados.length, true)
    cen.setUint16(28, nomeB.length, true)
    cen.setUint32(42, pos, true)
    central.push(new Uint8Array(cen.buffer), nomeB)
    pos += 30 + nomeB.length + dados.length
  }
  const tamCentral = central.reduce((s, b) => s + b.length, 0)
  const fim = new DataView(new ArrayBuffer(22))
  fim.setUint32(0, 0x06054b50, true)
  fim.setUint16(8, arquivos.length, true)
  fim.setUint16(10, arquivos.length, true)
  fim.setUint32(12, tamCentral, true)
  fim.setUint32(16, pos, true)
  return new Blob([...partes, ...central, new Uint8Array(fim.buffer)], {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  })
}

// Monta o arquivo .xlsx inteiro (devolve os bytes, sem baixar).
export function montarPlanilha(opcoes) {
  const nomeAba = esc((opcoes.aba || 'Lista').replace(/[\\/?*[\]:]/g, ' ').slice(0, 31))
  return zip([
    ['[Content_Types].xml', `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
</Types>`],
    ['_rels/.rels', `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`],
    ['xl/workbook.xml', `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="${nomeAba}" sheetId="1" r:id="rId1"/></sheets>
</workbook>`],
    ['xl/_rels/workbook.xml.rels', `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`],
    ['xl/styles.xml', ESTILOS],
    ['xl/worksheets/sheet1.xml', montarAba(opcoes)],
  ])
}

// Monta e baixa a planilha no computador.
export function baixarPlanilha(nomeArquivo, opcoes) {
  const url = URL.createObjectURL(montarPlanilha(opcoes))
  const a = document.createElement('a')
  a.href = url
  a.download = nomeArquivo
  a.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

// Data de hoje para o subtítulo: "09/11/2026 12:40".
export const agora = () =>
  new Date().toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
