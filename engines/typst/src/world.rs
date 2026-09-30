//! `World` do Typst sem sistema de arquivos: tudo o que o documento lê foi entregue pelo host.

use std::collections::HashMap;

use typst::diag::{FileError, FileResult};
use typst::foundations::{Bytes, Datetime, Duration};
use typst::syntax::package::PackageSpec;
use typst::syntax::{FileId, RootedPath, Source, VirtualPath, VirtualRoot};
use typst::text::{Font, FontBook};
use typst::utils::LazyHash;
use typst::{Library, LibraryExt, World};

pub struct LiafWorld {
    library: LazyHash<Library>,
    book: LazyHash<FontBook>,
    fonts: Vec<Font>,
    files: HashMap<FileId, Bytes>,
    sources: HashMap<FileId, Source>,
    main: FileId,
    today: Option<Datetime>,
}

impl LiafWorld {
    pub fn new() -> Self {
        let fonts = embedded_fonts();
        Self {
            library: LazyHash::new(Library::default()),
            book: LazyHash::new(FontBook::from_fonts(&fonts)),
            fonts,
            files: HashMap::new(),
            sources: HashMap::new(),
            main: id_for("/main.typ").expect("caminho fixo válido"),
            today: None,
        }
    }

    pub fn add_font(&mut self, data: Vec<u8>) -> usize {
        let added: Vec<Font> = Font::iter(Bytes::new(data)).collect();
        let n = added.len();
        self.fonts.extend(added);
        self.book = LazyHash::new(FontBook::from_fonts(&self.fonts));
        n
    }

    pub fn set_file(&mut self, path: &str, data: Vec<u8>) -> bool {
        let Some(id) = id_for(path) else { return false };
        self.sources.remove(&id);
        self.files.insert(id, Bytes::new(data));
        true
    }

    pub fn clear_files(&mut self) {
        self.files.clear();
        self.sources.clear();
    }

    pub fn prepare(&mut self, main: &str, today: Option<&str>) -> Result<(), String> {
        self.main = id_for(main).ok_or_else(|| format!("caminho inválido: {main}"))?;
        if !self.files.contains_key(&self.main) {
            return Err(format!("arquivo principal não enviado: {main}"));
        }
        self.today = match today {
            None => None,
            Some(text) => Some(parse_date(text).ok_or_else(|| format!("data inválida: {text}"))?),
        };
        Ok(())
    }

    pub fn line_column(&self, id: FileId, byte: usize) -> Option<(usize, usize)> {
        self.source(id).ok()?.lines().byte_to_line_column(byte)
    }
}

impl World for LiafWorld {
    fn library(&self) -> &LazyHash<Library> {
        &self.library
    }

    fn book(&self) -> &LazyHash<FontBook> {
        &self.book
    }

    fn main(&self) -> FileId {
        self.main
    }

    fn source(&self, id: FileId) -> FileResult<Source> {
        if let Some(source) = self.sources.get(&id) {
            return Ok(source.clone());
        }
        let bytes = self.file(id)?;
        let text = std::str::from_utf8(&bytes).map_err(|_| FileError::InvalidUtf8)?;
        Ok(Source::new(id, text.to_string()))
    }

    fn file(&self, id: FileId) -> FileResult<Bytes> {
        if let Some(bytes) = self.files.get(&id) {
            return Ok(bytes.clone());
        }
        if let VirtualRoot::Package(spec) = id.root() {
            // O motor não acessa a rede: pacotes chegam do host, embutidos no build.
            return Err(FileError::Other(Some(
                format!(
                    "pacote {spec} não embutido (o liafc build embute os pacotes citados no programa)"
                )
                .into(),
            )));
        }
        Err(FileError::NotFound(id.vpath().get_with_slash().into()))
    }

    fn font(&self, index: usize) -> Option<Font> {
        self.fonts.get(index).cloned()
    }

    fn today(&self, _offset: Option<Duration>) -> Option<Datetime> {
        self.today
    }
}

#[cfg(feature = "embedded-fonts")]
fn embedded_fonts() -> Vec<Font> {
    typst_assets::fonts().flat_map(|data| Font::iter(Bytes::new(data))).collect()
}

#[cfg(not(feature = "embedded-fonts"))]
fn embedded_fonts() -> Vec<Font> {
    Vec::new()
}

/// Caminho virtual -> FileId. "/x.typ" é do projeto; "@ns/nome:versão/x.typ" é um arquivo
/// de pacote (o host entrega os pacotes com esse prefixo).
fn id_for(path: &str) -> Option<FileId> {
    if let Some(rest) = path.strip_prefix('@') {
        let (spec, inner) = match rest.find(|c| c == '/').and_then(|ns_end| {
            rest[ns_end + 1..].find('/').map(|i| ns_end + 1 + i)
        }) {
            Some(split) => (&path[..split + 1], &rest[split..]),
            None => return None,
        };
        let spec: PackageSpec = spec.parse().ok()?;
        let vpath = VirtualPath::new(inner).ok()?;
        return Some(RootedPath::new(VirtualRoot::Package(spec), vpath).intern());
    }
    let vpath = VirtualPath::new(path).ok()?;
    Some(RootedPath::new(VirtualRoot::Project, vpath).intern())
}

fn parse_date(text: &str) -> Option<Datetime> {
    let mut parts = text.splitn(3, '-');
    let year = parts.next()?.parse().ok()?;
    let month = parts.next()?.parse().ok()?;
    let day = parts.next()?.parse().ok()?;
    Datetime::from_ymd(year, month, day)
}
