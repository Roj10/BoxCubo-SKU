# ============================================================
#  GERADOR DO "SEED" (os dados iniciais do sistema)
# ============================================================
#  Lê as peças tiradas da planilha da V2 (tools/v2_planilha.json) e monta
#  o arquivo backend/internal/store/seed.json, que é o ponto de partida
#  quando o sistema roda pela primeira vez. Só precisa rodar de novo se a
#  planilha de origem mudar:   python tools/gerar_seed.py

"""Gera backend/internal/store/seed.json a partir de tools/v2_planilha.json.

Uso:  python tools/gerar_seed.py
Cada linha da planilha: [grupo, classe, nome, onde_usado, qtd_por_maquina,
unidade, valor_unitario, fornecedor, estoque]
"""
import json
import pathlib

ROOT = pathlib.Path(__file__).resolve().parent.parent

GRUPOS = {
    "ELÉTRICA / ELETRÔNICA": "ELE",
    "ESTRUTURAL": "EST",
    "EMBALAGENS / OUTROS": "EMB",
}
CLASSES = {
    "RESISTOR": "RES", "CAPACITOR": "CAP", "LED": "LED", "DIODO": "DIO",
    "TRANSISTOR": "TRS", "OUTROS COMPONENTES": "CMP", "BARRA DE PINO": "BPN",
    "CONTROLADORES": "CTR", "CABEAMENTO": "CAB", "PCB": "PCB",
    "PESAGEM": "PES", "ADICIONAL CORREIOS": "PER", "METALÚRGICO": "MET",
    "ACABAMENTO": "ACA", "ADESIVOS": "ADE", "CASES": "CAS",
    "PARAFUSOS": "PAR", "EMBALAGENS": "EMB",
}
TIPOS = {"und": "peca", "rolo": "consumivel", "m": "consumivel", "serviço": "servico"}

MACHINES = [
    {"id": "v2", "sku": "MAQ-V2", "nome": "BoxCubo Estática V2", "apelido": "A base",
     "tecnologia": "Ultrassônico", "ativa": True},
    {"id": "v25", "sku": "MAQ-V25", "nome": "BoxCubo Estática V2.5", "apelido": "Revisão da V2",
     "tecnologia": "Ultrassônico", "ativa": False},
    {"id": "vision", "sku": "MAQ-VIS", "nome": "BoxCubo Vision", "apelido": "A inteligência",
     "tecnologia": "Câmera RGB + 3D com IA", "ativa": False},
    {"id": "dynamic", "sku": "MAQ-DYN", "nome": "BoxCubo Dynamic", "apelido": "A performance",
     "tecnologia": "Laser em esteira", "ativa": False},
    {"id": "eye", "sku": "MAQ-EYE", "nome": "BoxCubo Eye", "apelido": "A prova",
     "tecnologia": "Auditoria por foto e vídeo", "ativa": False},
]

KITS = [
    {"id": "ADDC", "nome": "Adicional Correios", "descricao": "Monitor, teclado, leitor 2D, nobreak e acessórios"},
]


def main():
    rows = json.loads((ROOT / "tools" / "v2_planilha.json").read_text(encoding="utf-8"))
    # finais descritivos (BC-ELE-RES-001 -> BC-ELE-RES-10); a mesma tabela é usada
    # pelo backend para migrar dados antigos. O texto de fornecedor ("FORNELL / ML")
    # é convertido em lista com código pelo backend na primeira vez que ele abre.
    descritivo = dict(json.loads(
        (ROOT / "backend" / "internal" / "store" / "sku_descritivo.json").read_text(encoding="utf-8")))
    contadores = {}
    parts, bom = [], []
    for grupo, classe, nome, onde, qtd, und, valor, forn, estoque in rows:
        g, c = GRUPOS[grupo], CLASSES[classe]
        contadores[(g, c)] = contadores.get((g, c), 0) + 1
        sku = f"BC-{g}-{c}-{contadores[(g, c)]:03d}"
        sku = descritivo.get(sku, sku).removeprefix("BC-")  # o "BC-" foi retirado dos SKUs
        parts.append({
            "sku": sku,
            "nome": nome.strip(),
            "grupo": grupo,
            "classe": classe,
            "unidade": und or "und",
            "tipo": TIPOS.get(und, "peca"),
            "fornecedor": forn,
            "valor_unitario": valor,
            "estoque": estoque or 0,
        })
        onde_lista = [o.strip() for o in onde.split(",") if o.strip()]
        kit = "ADDC" if "ADDC" in onde_lista else ""
        bom.append({
            "sku": sku,
            "quantidade": qtd if qtd is not None else 0,
            "onde_usado": [o for o in onde_lista if o != "ADDC"],
            "kit": kit,
        })

    seed = {
        "machines": MACHINES,
        "kits": KITS,
        "parts": parts,
        "bom": {"v2": bom},
        "legenda": {
            "PR": "Placa Raspberry", "PS": "Placa Sensor", "KM": "Kit Manutenção",
            "CS": "Cabo Sensor", "CF": "Cabo de força", "CL": "Cabo leitor (?)",
            "CR": "Cabo de rede (?)", "OS": "Sensores (?)", "BL": "Balança (?)",
            "FB": "Fixação balança (?)", "EMB": "Embalagem (?)",
            "EST": "Estrutural", "ACA": "Acabamento", "ADE": "Adesivo",
            "ESE": "Estrutural sensor", "ESR": "Estrutural Raspberry",
            "ESM": "Estrutural Monitor", "PTL": "Patola", "PFB": "Parafuso Balança",
            "PFE": "Parafuso Estrutura", "PFR": "Parafuso Raspberry",
            "PFS": "Parafuso Sensor", "FE": "Fixação estrutura",
            "FTB": "Fixação tampo balança", "BEM": "Embalagens e proteções",
        },
    }
    out = ROOT / "backend" / "internal" / "store" / "seed.json"
    out.write_text(json.dumps(seed, ensure_ascii=False, indent=1), encoding="utf-8")
    print(f"{len(parts)} peças gravadas em {out}")


if __name__ == "__main__":
    main()
