// ============================================================
//  MODELOS: o "formulário" de cada coisa que o sistema guarda
// ============================================================
//  Aqui não tem lógica, só a DEFINIÇÃO de cada tipo de informação, como as
//  colunas de uma planilha. Ex.: uma Peça tem SKU, nome, unidade, preço,
//  estoque, fornecedores e foto. Os textos entre crases, como `json:"sku"`,
//  dizem com que nome cada campo é gravado no arquivo e enviado para a tela.

// Package model define as entidades do sistema de SKU da BoxCubo.
package model

// Machine é um modelo de máquina vendido pela BoxCubo (V2, Vision, ...).
// MÁQUINA: um modelo vendido pela BoxCubo (Estática V2, Vision, Dynamic...).
// Guarda também o estoque mínimo: "manter peças para N máquinas".
type Machine struct {
	ID         string `json:"id"`
	SKU        string `json:"sku"`
	Nome       string `json:"nome"`
	Apelido    string `json:"apelido"`
	Tecnologia string `json:"tecnologia"`
	Ativa      bool   `json:"ativa"` // false = ainda sem lista de peças cadastrada
	// Estoque mínimo: manter peças para N máquinas deste modelo (0 = sem mínimo).
	MinimoMaquinas int  `json:"minimo_maquinas"`
	MinimoComKits  bool `json:"minimo_com_kits"` // itens dos kits também entram no mínimo
}

// Kit agrupa itens opcionais de uma máquina (ex.: Adicional Correios).
// KIT: um conjunto de itens OPCIONAIS da máquina.
// Ex.: o kit "Adicional Correios" (monitor, teclado, leitor 2D, nobreak).
type Kit struct {
	ID        string `json:"id"`
	Nome      string `json:"nome"`
	Descricao string `json:"descricao"`
}

// Part é uma peça, consumível ou serviço com SKU próprio.
// O SKU segue o formato BC-<GRUPO>-<CLASSE>-<NNN> e nunca muda,
// mesmo que nome, fornecedor ou preço mudem.
// PEÇA: cada item que entra numa máquina (resistor, balança, parafuso...).
// O SKU é o "RG" da peça: ex. ELE-RES-10 = Elétrica / Resistor / 10 Ω.
type Part struct {
	SKU     string `json:"sku"`
	Nome    string `json:"nome"`
	Grupo   string `json:"grupo"`
	Classe  string `json:"classe"`
	Unidade string `json:"unidade"` // und, rolo, m, serviço
	Tipo    string `json:"tipo"`    // peca, consumivel, servico
	// Fornecedores da peça, com o código do item em cada um; o primeiro é o principal.
	Fornecedores []Fornecimento `json:"fornecedores"`
	// FornecedorLegado é o texto livre antigo ("FORNELL / ML"), só para migrar dados.
	FornecedorLegado string   `json:"fornecedor,omitempty"`
	ValorUnitario    *float64 `json:"valor_unitario"` // nil = sem preço cadastrado
	Estoque          float64  `json:"estoque"`
	Imagem           string   `json:"imagem"` // URL da foto (enviada pela API de imagens); "" = sem foto
}

// Fornecimento é um fornecedor da peça e o código do item nele.
// FORNECIMENTO: de quem a peça é comprada e qual o código dela naquela loja.
// Ex.: {Nome: "1001 Parafusos", Codigo: "2241"}.
type Fornecimento struct {
	Nome   string `json:"nome"`
	Codigo string `json:"codigo"` // código/referência do item no fornecedor (opcional)
}

// BOMItem liga uma peça a uma máquina (lista de materiais).
// ITEM DA LISTA DE MATERIAIS (BOM): liga uma peça a uma máquina.
// Ex.: "a V2 usa 10 sensores ultrassônicos, na Placa Sensor (PS)".
type BOMItem struct {
	SKU        string   `json:"sku"`
	Quantidade float64  `json:"quantidade"` // por máquina
	OndeUsado  []string `json:"onde_usado"` // códigos da legenda (PR, PS, ...)
	Kit        string   `json:"kit"`        // "" = item padrão; senão ID do kit
}

// BOMLine é um item da lista já combinado com os dados da peça.
// LINHA DA LISTA PRONTA PARA A TELA: o item da lista + os dados da peça +
// o custo por máquina + o estoque mínimo (já calculados).
type BOMLine struct {
	BOMItem
	Peca            Part     `json:"peca"`
	CustoPorMaquina *float64 `json:"custo_por_maquina"`
	Minimo          float64  `json:"minimo"` // estoque mínimo da peça (somando todas as máquinas)
}

// ReposicaoLine é uma peça abaixo do estoque mínimo.
// LINHA DA TELA "REPOSIÇÃO": uma peça abaixo do estoque mínimo e quanto
// comprar para voltar ao mínimo.
type ReposicaoLine struct {
	SKU          string         `json:"sku"`
	Nome         string         `json:"nome"`
	Unidade      string         `json:"unidade"`
	Estoque      float64        `json:"estoque"`
	Minimo       float64        `json:"minimo"`
	Sugestao     float64        `json:"sugestao"` // quanto comprar para voltar ao mínimo
	ValorUnit    *float64       `json:"valor_unitario"`
	Custo        *float64       `json:"custo"`
	Fornecedores []Fornecimento `json:"fornecedores"`
	Imagem       string         `json:"imagem"`
	Zerado       bool           `json:"zerado"`
	Maquinas     []string       `json:"maquinas"` // modelos cujo mínimo exige esta peça
}

// Classe descreve um prefixo de SKU disponível para novas peças.
// CLASSE: um "prefixo" de SKU disponível, ex. ELE-RES (resistores).
// Usado no formulário de peça nova para escolher o tipo.
type Classe struct {
	Prefixo string `json:"prefixo"` // ex.: BC-ELE-RES
	Grupo   string `json:"grupo"`
	Classe  string `json:"classe"`
}

// PlanLine é a necessidade de compra de uma peça para um lote.
// LINHA DO "PLANEJAR LOTE": para uma peça, quanto precisa para o lote,
// quanto tem em estoque e quanto falta comprar.
type PlanLine struct {
	SKU          string         `json:"sku"`
	Nome         string         `json:"nome"`
	Unidade      string         `json:"unidade"`
	Kit          string         `json:"kit"`
	PorMaquina   float64        `json:"por_maquina"`
	Necessario   float64        `json:"necessario"`
	ComMargem    float64        `json:"com_margem"`
	Estoque      float64        `json:"estoque"`
	Comprar      float64        `json:"comprar"`
	ValorUnit    *float64       `json:"valor_unitario"`
	CustoCompra  *float64       `json:"custo_compra"`
	Fornecedores []Fornecimento `json:"fornecedores"`
	Imagem       string         `json:"imagem"`
}

// Tipos de movimentação de estoque.
// TIPOS DE MOVIMENTAÇÃO DE ESTOQUE (o "motivo" de cada entrada/saída).
const (
	MovSaidaMaquina = "saida_maquina" // máquina montada/vendida: baixa a lista de peças
	MovSaidaPecas   = "saida_pecas"   // peças avulsas enviadas (manutenção, reposição)
	MovEntrada      = "entrada"       // material comprado chegou
	MovEstorno      = "estorno"       // desfaz outra movimentação
)

// StockChange é um pedido de alteração de estoque (Delta < 0 = saída).
// PEDIDO DE MUDANÇA DE ESTOQUE: "mexer X unidades na peça Y".
// Número negativo = sai do estoque; positivo = entra.
type StockChange struct {
	SKU   string  `json:"sku"`
	Delta float64 `json:"delta"`
}

// MovementItem é o efeito real de uma movimentação numa peça.
// O QUE REALMENTE ACONTECEU COM UMA PEÇA numa movimentação: quanto foi
// pedido, quanto saiu de fato, estoque antes/depois e quanto faltou.
type MovementItem struct {
	SKU           string  `json:"sku"`
	Nome          string  `json:"nome"`
	Unidade       string  `json:"unidade"`
	Solicitado    float64 `json:"solicitado"`  // quanto se pediu para mover (positivo)
	Movimentado   float64 `json:"movimentado"` // quanto realmente mudou (negativo = saiu)
	EstoqueAntes  float64 `json:"estoque_antes"`
	EstoqueDepois float64 `json:"estoque_depois"`
	Falta         float64 `json:"falta"` // saída sem estoque suficiente: o que ficou faltando
}

// Movement é um registro do histórico de estoque.
// MOVIMENTAÇÃO: um registro do histórico (como uma linha no caderno do
// estoque). Ex.: "MOV-000003, saída de 2 máquinas V2, séries BC2-0451 e
// BC2-0452, cliente Correios". Nunca é apagada; se foi errada, é ESTORNADA.
type Movement struct {
	ID           string         `json:"id"` // MOV-000001
	Data         string         `json:"data"`
	Tipo         string         `json:"tipo"`
	MachineID    string         `json:"machine_id,omitempty"`
	Quantidade   int            `json:"quantidade,omitempty"` // máquinas
	NumerosSerie []string       `json:"numeros_serie,omitempty"`
	Kits         []string       `json:"kits,omitempty"`
	Cliente      string         `json:"cliente,omitempty"`
	Observacao   string         `json:"observacao,omitempty"`
	ComFalta     bool           `json:"com_falta"`
	Estornado    bool           `json:"estornado"`
	EstornadoPor string         `json:"estornado_por,omitempty"`
	EstornoDe    string         `json:"estorno_de,omitempty"`
	Itens        []MovementItem `json:"itens"`
}

// Shortage é uma peça sem estoque suficiente para uma saída.
// FALTA: peça sem estoque suficiente para uma saída (usada no pop-up
// "Não há peças suficientes").
type Shortage struct {
	SKU        string  `json:"sku"`
	Nome       string  `json:"nome"`
	Unidade    string  `json:"unidade"`
	Necessario float64 `json:"necessario"`
	Estoque    float64 `json:"estoque"`
	Falta      float64 `json:"falta"`
	Imagem     string  `json:"imagem"`
}

// Plan é o resultado do planejamento de produção de um lote.
// RESULTADO COMPLETO DO "PLANEJAR LOTE": todas as linhas + totais.
type Plan struct {
	MachineID     string     `json:"machine_id"`
	Quantidade    int        `json:"quantidade"`
	Margem        float64    `json:"margem"`
	Kits          []string   `json:"kits"`
	Linhas        []PlanLine `json:"linhas"`
	CustoLote     float64    `json:"custo_lote"`      // tudo, ignorando estoque
	CustoCompra   float64    `json:"custo_compra"`    // só o que falta comprar
	ItensFaltando int        `json:"itens_faltando"`  // linhas com compra > 0
	ItensSemPreco int        `json:"itens_sem_preco"` // linhas sem valor unitário
}
