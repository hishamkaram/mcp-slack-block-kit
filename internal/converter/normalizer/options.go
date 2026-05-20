package normalizer

// Options gates the opt-in repairs in the normalizer pipeline. The vast
// majority of repairs run unconditionally because they're safe — every
// option here exists to mark a repair as "safe by default *off* because
// the trade-off involves the broadcast-safety contract or false-positive
// risk on real prose".
type Options struct {
	// DecodeHTMLEntities enables the whitelisted entity decoder
	// (&amp; &lt; &gt; &quot; &apos; + numeric character references).
	// Default false. Opt-in because the resulting `&`/`<`/`>` chars
	// will re-escape through the converter's sanitizeBroadcasts pass —
	// the outcome is safe, but worth being explicit about for any
	// caller that audits broadcast-token paths.
	DecodeHTMLEntities bool

	// RepairMismatchedEmphasis enables the paragraph-level asterisk
	// balancer (V6 in the LLM-input recovery catalog). Default false
	// because the algorithm is the trickiest in the catalog and
	// false-positives on real corpora corrupt prose. Enable after
	// reviewing the catalog doc and confirming your input shape.
	RepairMismatchedEmphasis bool
}
