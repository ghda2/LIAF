//! Motor de documentos da LIAF.
//!
//! Compila o Typst para `wasm32-wasip1` e expõe uma ABI mínima, chamada pelo runtime Go
//! (`pkg/typst`) através do wazero. O estado vive dentro da instância: fontes carregadas uma vez,
//! arquivos virtuais trocados a cada renderização e a memoização do Typst (comemo) aproveitada
//! entre chamadas.
//!
//! ABI (todas as funções usam memória linear; ponteiros e tamanhos em u32):
//!
//! - `liaf_alloc(len) -> ptr` / `liaf_free(ptr, len)`: buffers para o host escrever entradas.
//! - `liaf_add_font(ptr, len) -> n`: registra as faces de um arquivo de fonte; devolve quantas.
//! - `liaf_set_file(path_ptr, path_len, data_ptr, data_len)`: cria/substitui um arquivo virtual.
//! - `liaf_clear_files()`: apaga os arquivos virtuais.
//! - `liaf_render(req_ptr, req_len) -> ok`: compila conforme o pedido JSON (ver [`Request`]).
//! - `liaf_result_ptr/len()`: JSON com diagnósticos e metadados da última renderização.
//! - `liaf_output_count()`, `liaf_output_ptr/len(i)`: bytes gerados (um PDF ou um PNG por página).
//! - `liaf_evict(max_age)`: descarta memoização antiga (limita a memória num servidor).

mod layout;
mod world;

use std::cell::RefCell;

use serde::{Deserialize, Serialize};
use typst::diag::{Severity, SourceDiagnostic};
use typst::foundations::Smart;
use typst_layout::PagedDocument;
use typst::WorldExt;
use typst_pdf::{PdfOptions, PdfStandard, PdfStandards};

use world::LiafWorld;

thread_local! {
    static STATE: RefCell<State> = RefCell::new(State::new());
}

struct State {
    world: LiafWorld,
    result: Vec<u8>,
    outputs: Vec<Vec<u8>>,
}

impl State {
    fn new() -> Self {
        Self { world: LiafWorld::new(), result: Vec::new(), outputs: Vec::new() }
    }
}

/// Pedido de renderização enviado pelo host.
#[derive(Deserialize)]
struct Request {
    /// Arquivo principal entre os arquivos virtuais (ex.: "/main.typ").
    main: String,
    /// "pdf" (padrão), "png" ou "check" (só compila e devolve diagnósticos e layout).
    #[serde(default)]
    format: Option<String>,
    /// Resolução do PNG em pixels por polegada (padrão 144).
    #[serde(default)]
    ppi: Option<f32>,
    /// Padrões PDF exigidos, na grafia do Typst: "a-2b", "ua-1", "1.7"...
    #[serde(default)]
    standards: Vec<String>,
    /// Identificador estável do documento; torna o /ID do PDF determinístico.
    #[serde(default)]
    ident: Option<String>,
    /// Data devolvida por `datetime.today()`, "AAAA-MM-DD". Sem ela, `today()` falha: o
    /// documento continua determinístico.
    #[serde(default)]
    today: Option<String>,
    /// Desliga o PDF marcado (acessível), que é o padrão.
    #[serde(default)]
    untagged: bool,
    /// Inclui o relatório de layout (caixas, conteúdo fora da página) na resposta.
    #[serde(default)]
    layout: bool,
}

#[derive(Serialize)]
struct Response {
    ok: bool,
    diagnostics: Vec<Diag>,
    pages: Vec<PageInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    layout: Option<Vec<layout::PageLayout>>,
}

#[derive(Serialize)]
struct PageInfo {
    width_pt: f64,
    height_pt: f64,
}

#[derive(Serialize)]
struct Diag {
    severity: &'static str,
    message: String,
    file: Option<String>,
    line: Option<usize>,
    column: Option<usize>,
    hints: Vec<String>,
}

fn render(state: &mut State, raw: &[u8]) -> Response {
    state.outputs.clear();
    let req: Request = match serde_json::from_slice(raw) {
        Ok(req) => req,
        Err(err) => return fail(format!("pedido inválido: {err}")),
    };
    if let Err(msg) = state.world.prepare(&req.main, req.today.as_deref()) {
        return fail(msg);
    }

    let compiled = typst::compile::<PagedDocument>(&state.world);
    let mut diagnostics: Vec<Diag> =
        compiled.warnings.iter().map(|d| diag(&state.world, d)).collect();
    let document = match compiled.output {
        Ok(doc) => doc,
        Err(errors) => {
            diagnostics.extend(errors.iter().map(|d| diag(&state.world, d)));
            return Response { ok: false, diagnostics, pages: Vec::new(), layout: None };
        }
    };
    let pages = document
        .pages()
        .iter()
        .map(|p| PageInfo { width_pt: p.frame.width().to_pt(), height_pt: p.frame.height().to_pt() })
        .collect();

    match req.format.as_deref().unwrap_or("pdf") {
        "check" => {}
        "png" => {
            let ppi = req.ppi.unwrap_or(144.0);
            let opts = typst_render::RenderOptions {
                pixel_per_pt: typst::utils::Scalar::new((ppi / 72.0) as f64),
                render_bleed: false,
            };
            for page in document.pages() {
                let pixmap = typst_render::render(page, &opts);
                match pixmap.encode_png() {
                    Ok(png) => state.outputs.push(png),
                    Err(err) => return fail(format!("falha ao gerar PNG: {err}")),
                }
            }
        }
        "pdf" => {
            let standards = match parse_standards(&req.standards) {
                Ok(s) => s,
                Err(msg) => return fail(msg),
            };
            let options = PdfOptions {
                ident: req.ident.map(Smart::Custom).unwrap_or(Smart::Auto),
                creator: Smart::Custom(Some("LIAF".into())),
                timestamp: None,
                page_ranges: None,
                standards,
                tagged: !req.untagged,
                pretty: false,
            };
            match typst_pdf::pdf(&document, &options) {
                Ok(bytes) => state.outputs.push(bytes),
                Err(errors) => {
                    diagnostics.extend(errors.iter().map(|d| diag(&state.world, d)));
                    return Response { ok: false, diagnostics, pages, layout: None };
                }
            }
        }
        other => return fail(format!("formato desconhecido: {other} (use pdf, png ou check)")),
    }
    let layout = req.layout.then(|| layout::report(document.pages()));
    Response { ok: true, diagnostics, pages, layout }
}

fn parse_standards(names: &[String]) -> Result<PdfStandards, String> {
    let mut list = Vec::with_capacity(names.len());
    for name in names {
        let quoted = serde_json::Value::String(name.clone());
        let standard: PdfStandard = serde_json::from_value(quoted)
            .map_err(|_| format!("padrão PDF desconhecido: {name}"))?;
        list.push(standard);
    }
    PdfStandards::new(&list).map_err(|err| err.message().to_string())
}

fn diag(world: &LiafWorld, d: &SourceDiagnostic) -> Diag {
    let mut out = Diag {
        severity: if d.severity == Severity::Error { "error" } else { "warning" },
        message: d.message.to_string(),
        file: None,
        line: None,
        column: None,
        hints: d.hints.iter().map(|h| h.v.to_string()).collect(),
    };
    if let (Some(id), Some(range)) = (d.span.id(), world.range(d.span)) {
        out.file = Some(id.vpath().get_with_slash().to_string());
        if let Some((line, col)) = world.line_column(id, range.start) {
            out.line = Some(line + 1);
            out.column = Some(col + 1);
        }
    }
    out
}

fn fail(message: String) -> Response {
    Response {
        ok: false,
        diagnostics: vec![Diag {
            severity: "error",
            message,
            file: None,
            line: None,
            column: None,
            hints: Vec::new(),
        }],
        pages: Vec::new(),
        layout: None,
    }
}

// ---- ABI ------------------------------------------------------------------------------------

/// # Safety
/// O host deve devolver o buffer com `liaf_free` usando o mesmo tamanho.
#[no_mangle]
pub extern "C" fn liaf_alloc(len: u32) -> *mut u8 {
    let mut buf = Vec::<u8>::with_capacity(len as usize);
    let ptr = buf.as_mut_ptr();
    std::mem::forget(buf);
    ptr
}

/// # Safety
/// `ptr` precisa ter vindo de `liaf_alloc(len)`.
#[no_mangle]
pub unsafe extern "C" fn liaf_free(ptr: *mut u8, len: u32) {
    drop(Vec::from_raw_parts(ptr, 0, len as usize));
}

unsafe fn slice<'a>(ptr: *const u8, len: u32) -> &'a [u8] {
    if len == 0 {
        return &[];
    }
    std::slice::from_raw_parts(ptr, len as usize)
}

/// # Safety
/// `ptr..ptr+len` precisa estar na memória linear.
#[no_mangle]
pub unsafe extern "C" fn liaf_add_font(ptr: *const u8, len: u32) -> u32 {
    let data = slice(ptr, len).to_vec();
    STATE.with(|s| s.borrow_mut().world.add_font(data) as u32)
}

/// # Safety
/// Os dois intervalos precisam estar na memória linear.
#[no_mangle]
pub unsafe extern "C" fn liaf_set_file(
    path_ptr: *const u8,
    path_len: u32,
    data_ptr: *const u8,
    data_len: u32,
) -> u32 {
    let Ok(path) = std::str::from_utf8(slice(path_ptr, path_len)) else { return 0 };
    let data = slice(data_ptr, data_len).to_vec();
    STATE.with(|s| s.borrow_mut().world.set_file(path, data)) as u32
}

#[no_mangle]
pub extern "C" fn liaf_clear_files() {
    STATE.with(|s| s.borrow_mut().world.clear_files());
}

/// # Safety
/// `ptr..ptr+len` precisa estar na memória linear.
#[no_mangle]
pub unsafe extern "C" fn liaf_render(ptr: *const u8, len: u32) -> u32 {
    let raw = slice(ptr, len);
    STATE.with(|s| {
        let mut state = s.borrow_mut();
        let response = render(&mut state, raw);
        let ok = response.ok;
        state.result = serde_json::to_vec(&response).unwrap_or_default();
        ok as u32
    })
}

#[no_mangle]
pub extern "C" fn liaf_result_ptr() -> *const u8 {
    STATE.with(|s| s.borrow().result.as_ptr())
}

#[no_mangle]
pub extern "C" fn liaf_result_len() -> u32 {
    STATE.with(|s| s.borrow().result.len() as u32)
}

#[no_mangle]
pub extern "C" fn liaf_output_count() -> u32 {
    STATE.with(|s| s.borrow().outputs.len() as u32)
}

#[no_mangle]
pub extern "C" fn liaf_output_ptr(i: u32) -> *const u8 {
    STATE.with(|s| s.borrow().outputs.get(i as usize).map_or(std::ptr::null(), |o| o.as_ptr()))
}

#[no_mangle]
pub extern "C" fn liaf_output_len(i: u32) -> u32 {
    STATE.with(|s| s.borrow().outputs.get(i as usize).map_or(0, |o| o.len() as u32))
}

#[no_mangle]
pub extern "C" fn liaf_evict(max_age: u32) {
    comemo::evict(max_age as usize);
}
