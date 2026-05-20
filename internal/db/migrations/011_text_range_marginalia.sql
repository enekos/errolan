-- Text-range marginalia: a comment can target a specific text selection
-- inside an anchor element, not just the anchor as a whole. We follow the
-- W3C Annotation TextQuoteSelector + TextPositionSelector pattern so the
-- SDK can re-anchor robustly when the host page mutates:
--
--   range_quote   exact selected text (used for primary lookup)
--   range_prefix  up to 32 chars immediately before the selection
--   range_suffix  up to 32 chars immediately after the selection
--   range_start   character offset within the anchor element (best-effort)
--   range_end     character offset within the anchor element (best-effort)
--
-- All five fields are optional; legacy paragraph-only anchors leave them empty.
ALTER TABLE comments ADD COLUMN range_quote  TEXT NOT NULL DEFAULT '';
ALTER TABLE comments ADD COLUMN range_prefix TEXT NOT NULL DEFAULT '';
ALTER TABLE comments ADD COLUMN range_suffix TEXT NOT NULL DEFAULT '';
ALTER TABLE comments ADD COLUMN range_start  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE comments ADD COLUMN range_end    INTEGER NOT NULL DEFAULT 0;
