//! Relatório de layout (issue #037): percorre as páginas montadas pelo Typst e mede onde o
//! conteúdo caiu. Serve para um modelo só de texto conferir o resultado sem ver a imagem, e
//! para os avisos W_LAYOUT_* do `liafc doc check`.

use serde::Serialize;
use typst::layout::{Abs, Frame, FrameItem, Point, Transform};
use typst::visualize::Paint;
use typst_layout::Page;

/// Tolerância para arredondamento de posições, em pontos.
const EPS: f64 = 0.5;

#[derive(Serialize)]
pub struct PageLayout {
    pub width_pt: f64,
    pub height_pt: f64,
    /// Caixa que envolve todo o conteúdo visível [x0, y0, x1, y1], ou null se a página está
    /// vazia. Fundos de página cheios (retângulo do tamanho da página) não contam.
    pub content: Option<[f64; 4]>,
    pub texts: usize,
    pub images: usize,
    pub shapes: usize,
    /// Elementos que passam da borda da página (conteúdo cortado na impressão).
    pub overflow: Vec<Overflow>,
    /// Textos com contraste abaixo do mínimo WCAG AA contra o fundo desenhado atrás deles.
    pub low_contrast: Vec<LowContrast>,
}

#[derive(Serialize)]
pub struct LowContrast {
    pub text: String,
    /// Razão de contraste (1 a 21). WCAG AA: 4,5 para texto normal, 3 para texto grande.
    pub ratio: f64,
    pub minimum: f64,
    pub color: String,
    pub background: String,
    pub size_pt: f64,
}

#[derive(Serialize)]
pub struct Overflow {
    pub kind: &'static str,
    pub text: Option<String>,
    pub bbox: [f64; 4],
    /// Quanto passou da borda, no pior lado, em pontos.
    pub by_pt: f64,
}

struct Acc {
    w: f64,
    h: f64,
    content: Option<[f64; 4]>,
    texts: usize,
    images: usize,
    shapes: usize,
    overflow: Vec<Overflow>,
    /// Fundos sólidos já desenhados (caixa, cor), em ordem de pintura.
    fills: Vec<([f64; 4], [f64; 3])>,
    page_bg: [f64; 3],
    low_contrast: Vec<LowContrast>,
}

pub fn report(pages: &[Page]) -> Vec<PageLayout> {
    pages
        .iter()
        .map(|page| {
            let size = page.frame.size();
            let mut acc = Acc {
                w: size.x.to_pt(),
                h: size.y.to_pt(),
                content: None,
                texts: 0,
                images: 0,
                shapes: 0,
                overflow: Vec::new(),
                fills: Vec::new(),
                page_bg: solid(page.fill_or_white().as_ref()).unwrap_or([1.0, 1.0, 1.0]),
                low_contrast: Vec::new(),
            };
            walk(&page.frame, Transform::identity(), &mut acc);
            PageLayout {
                width_pt: acc.w,
                height_pt: acc.h,
                content: acc.content,
                texts: acc.texts,
                images: acc.images,
                shapes: acc.shapes,
                overflow: acc.overflow,
                low_contrast: acc.low_contrast,
            }
        })
        .collect()
}

fn walk(frame: &Frame, ts: Transform, acc: &mut Acc) {
    for (pos, item) in frame.items() {
        let local = ts.pre_concat(Transform::translate(pos.x, pos.y));
        match item {
            FrameItem::Group(group) => {
                let inner = local.pre_concat(group.transform);
                if group.clip.is_some() {
                    // Conteúdo recortado de propósito (ex.: foto num círculo): conta a caixa
                    // do grupo, não o que ficou escondido dentro dele.
                    let s = group.frame.size();
                    let b = bbox(inner, 0.0, 0.0, s.x.to_pt(), s.y.to_pt());
                    add(acc, b, "grupo", None);
                } else {
                    walk(&group.frame, inner, acc);
                }
            }
            FrameItem::Text(text) => {
                acc.texts += 1;
                let size = text.size.to_pt();
                let b = bbox(local, 0.0, -0.8 * size, text.width().to_pt(), 0.25 * size);
                let snippet: String = text.text.chars().take(48).collect();
                if !snippet.trim().is_empty() {
                    check_contrast(acc, b, text, &snippet);
                }
                add(acc, b, "texto", Some(snippet));
            }
            FrameItem::Shape(shape, _) => {
                acc.shapes += 1;
                let r = shape.bbox(true);
                let b = bbox(local, r.min.x.to_pt(), r.min.y.to_pt(), r.max.x.to_pt(), r.max.y.to_pt());
                // Fundo que cobre a página inteira (ou quase) é decoração, não conteúdo.
                let full = (b[2] - b[0]) >= acc.w * 0.98 && (b[3] - b[1]) >= acc.h * 0.98;
                let tall_band = (b[3] - b[1]) >= acc.h * 0.98 && b[1] <= EPS;
                if !full && !tall_band {
                    add(acc, b, "forma", None);
                }
                if let Some(rgb) = solid(shape.fill.as_ref()) {
                    acc.fills.push((b, rgb));
                }
            }
            FrameItem::Image(_, size, _) => {
                acc.images += 1;
                let b = bbox(local, 0.0, 0.0, size.x.to_pt(), size.y.to_pt());
                add(acc, b, "imagem", None);
            }
            FrameItem::Link(..) | FrameItem::Tag(..) => {}
        }
    }
}

fn bbox(ts: Transform, x0: f64, y0: f64, x1: f64, y1: f64) -> [f64; 4] {
    let corners = [(x0, y0), (x1, y0), (x0, y1), (x1, y1)];
    let mut out = [f64::MAX, f64::MAX, f64::MIN, f64::MIN];
    for (x, y) in corners {
        let p = Point::new(Abs::pt(x), Abs::pt(y)).transform(ts);
        out[0] = out[0].min(p.x.to_pt());
        out[1] = out[1].min(p.y.to_pt());
        out[2] = out[2].max(p.x.to_pt());
        out[3] = out[3].max(p.y.to_pt());
    }
    out
}

fn add(acc: &mut Acc, b: [f64; 4], kind: &'static str, text: Option<String>) {
    acc.content = Some(match acc.content {
        None => b,
        Some(c) => [c[0].min(b[0]), c[1].min(b[1]), c[2].max(b[2]), c[3].max(b[3])],
    });
    let by = [-b[0], -b[1], b[2] - acc.w, b[3] - acc.h].into_iter().fold(0.0, f64::max);
    if by > EPS && acc.overflow.len() < 20 {
        acc.overflow.push(Overflow { kind, text, bbox: b.map(round), by_pt: round(by) });
    }
}

fn round(v: f64) -> f64 {
    (v * 10.0).round() / 10.0
}

/// Cor sólida opaca (alfa >= 0,5) em sRGB de 0 a 1; gradientes e padrões ficam de fora.
fn solid(paint: Option<&Paint>) -> Option<[f64; 3]> {
    match paint {
        Some(Paint::Solid(color)) => {
            let c = color.to_rgb();
            (c.alpha >= 0.5).then(|| [c.red as f64, c.green as f64, c.blue as f64])
        }
        _ => None,
    }
}

fn check_contrast(acc: &mut Acc, b: [f64; 4], text: &typst::text::TextItem, snippet: &str) {
    let Some(fg) = solid(Some(&text.fill)) else { return };
    let (cx, cy) = ((b[0] + b[2]) / 2.0, (b[1] + b[3]) / 2.0);
    let bg = acc
        .fills
        .iter()
        .rev()
        .find(|(r, _)| r[0] <= cx && cx <= r[2] && r[1] <= cy && cy <= r[3])
        .map(|(_, c)| *c)
        .unwrap_or(acc.page_bg);
    let ratio = contrast(fg, bg);
    let size = text.size.to_pt();
    let bold = text.font.font().info().variant.weight.to_number() >= 700;
    // WCAG: texto grande é >= 18 pt, ou >= 14 pt em negrito.
    let minimum = if size >= 18.0 || (bold && size >= 14.0) { 3.0 } else { 4.5 };
    if ratio + 0.005 < minimum && acc.low_contrast.len() < 20 {
        acc.low_contrast.push(LowContrast {
            text: snippet.to_string(),
            ratio: (ratio * 100.0).round() / 100.0,
            minimum,
            color: hex(fg),
            background: hex(bg),
            size_pt: round(size),
        });
    }
}

fn luminance(c: [f64; 3]) -> f64 {
    let ch = |v: f64| if v <= 0.03928 { v / 12.92 } else { ((v + 0.055) / 1.055).powf(2.4) };
    0.2126 * ch(c[0]) + 0.7152 * ch(c[1]) + 0.0722 * ch(c[2])
}

fn contrast(a: [f64; 3], b: [f64; 3]) -> f64 {
    let (la, lb) = (luminance(a), luminance(b));
    (la.max(lb) + 0.05) / (la.min(lb) + 0.05)
}

fn hex(c: [f64; 3]) -> String {
    let [r, g, b] = c.map(|v| (v.clamp(0.0, 1.0) * 255.0).round() as u8);
    format!("#{r:02x}{g:02x}{b:02x}")
}
