# HAL: The Hypertext Application Language

**Being a SOCRATIC INQUIRY into the Nature of `application/hal+json`, in which THE NOVICE poses questions to ROY T. FIELDING (or at least to the spirit of his Dissertation, which haunts all JSON endpoints that dare call themselves RESTful), and receives answers that are sometimes encouraging, sometimes devastating, and once genuinely sad.**

_This dialogue was reconstructed from HTTP response headers found in the `Link` field of an abandoned HAL+JSON API. The API had twelve consumers. Eleven of them were hardcoding URLs. The twelfth was a browser. The browser was the only one that didn't break when the API changed. Nobody learned anything from this._

---

## The First Question: What Is HAL?

**THE NOVICE asked: "Master, I have heard of HAL. They say it brings hypermedia to JSON. Is this true?"**

It is... partially true. And partial truth is the most dangerous kind, because it lets you stop thinking right when thinking becomes important.

HAL — the Hypertext Application Language — adds two reserved properties to a JSON object: `_links` and `_embedded`. That is the entire specification. Two underscored keys. A draft RFC that never graduated. And yet it is the most widely adopted hypermedia JSON format in existence, which tells you something about the market for hypermedia JSON formats, and also about the power of being simple enough to explain in a single sentence.

```json
{
  "_links": {
    "self": { "href": "/books/1" },
    "author": { "href": "/authors/3", "title": "Mike Amundsen" }
  },
  "title": "Building Hypermedia APIs",
  "year": 2011
}
```

The `_links` object maps link relation types to link objects. The relation types come from the [IANA Link Relations Registry](https://www.iana.org/assignments/link-relations/link-relations.xhtml) — the same registry that gives meaning to `<link rel="stylesheet">` and `<a rel="nofollow">`. The link objects carry an `href` and optional metadata: `title`, `type`, `templated`, `deprecation`, `name`, `profile`, `hreflang`.

This is good. This is genuinely good. A JSON response that tells you where to go next is infinitely better than a JSON response that tells you nothing and expects you to consult a Swagger document that was last updated during the previous equinox.

**THE NOVICE said: "Then HAL solves the problem. JSON can be hypermedia."**

Sit down.

---

## The Second Question: What Does HAL Give You?

**THE NOVICE asked: "What can a client do with a HAL response?"**

It can _navigate_. That is what `_links` provides. The client receives a representation, sees which resources are related, and follows links to reach them. No URL construction. No out-of-band route table. No generated API client that breaks when you rename a path segment.

```json
{
  "_links": {
    "self":     { "href": "/catalog" },
    "books":    { "href": "/books", "title": "All Books" },
    "search":   { "href": "/books?q={query}", "templated": true }
  },
  "name": "Hypermedia Bookshop"
}
```

The client discovers `/books` and `/books?q={query}` from the response itself. If the server moves the books to `/api/v2/books`, the client doesn't break — it reads the new `_links` and follows them. This is HATEOAS applied to JSON. The representation drives the navigation.

HAL also gives you `_embedded`, which includes sub-resources inline:

```json
{
  "_links": { "self": { "href": "/books" } },
  "_embedded": {
    "books": [
      {
        "_links": { "self": { "href": "/books/1" }, "author": { "href": "/authors/3" } },
        "title": "Building Hypermedia APIs",
        "year": 2011
      }
    ]
  }
}
```

This reduces round-trips. Instead of fetching `/books` and then fetching each book, the server includes the books inline. The client renders them immediately. Each embedded resource carries its own `_links`, so the client can navigate deeper without a separate request.

**THE NOVICE said: "This is excellent. Navigation and embedding. What more does one need?"**

I am going to answer your question. But first I need you to look at an HTML page. Any HTML page.

---

## The Third Question: What Does HAL Not Give You?

**THE NOVICE looked at an HTML page. THE NOVICE saw links. THE NOVICE saw forms. THE NOVICE saw buttons with methods and actions and input fields.**

**THE NOVICE said: "Oh."**

Yes. "Oh."

HTML gives you `<a>` tags (navigation) AND `<form>` tags (mutation). A form tells the client: here is a thing you can do, here is the method to use, here is the URL to send it to, and here are the fields you need to fill in. The form is a complete instruction set for a state transition. The client doesn't need documentation. The client doesn't need a schema. The client renders the form, the user fills it in, the browser submits it. THE REPRESENTATION IS THE API.

Now look at HAL again.

Where is the form?

**THE NOVICE looked at the HAL response.**

**THE NOVICE said: "There is no form."**

There is no form. HAL has `_links`. It does not have `_actions`, or `_forms`, or `_operations`, or `_templates`, or anything that tells the client what it can DO to a resource. You can navigate. You cannot mutate. You can find the book. You cannot edit the book. You can see the author. You cannot create a new author. The map shows the territory but not the roads you are allowed to build.

This is the fundamental gap. HAL solves half of HATEOAS — the navigation half. The `<a>` tag half. It does not solve the other half — the `<form>` tag half. And the `<form>` tag half is where the interesting work happens, because anyone can follow links, but knowing which state transitions are available _without out-of-band information_ — that is the engine of application state. That is what makes the web work. That is what makes a browser able to operate a banking application without a BankingApplicationSDK.

**THE NOVICE was quiet for a while.**

**THE NOVICE said: "So a HAL client still needs to know how to mutate resources?"**

Yes. A HAL client knows _where things are_ from `_links`. But it must know _what it can do to them_ from documentation, convention, or prior knowledge. The URL came from the server. The method, the request body, the content type, the required fields — those came from the developer's head, or from a README, or from Kevin's Slack message that said "just POST to it with JSON, it'll work."

This is better than a bare JSON API where the client also has to know all the URLs. But it is not HATEOAS. It is HAEOS at best — Hypermedia As the Engine of _some_ State, specifically the navigational state, but not the transactional state. Like a tourist who has an excellent map but cannot read the signs that say "OPEN" and "CLOSED" and "PUSH" and "PULL."

---

## The Fourth Question: What About the Others?

**THE NOVICE asked: "Are there formats that solve the whole problem?"**

Several tried.

[**Siren**](https://github.com/kevinswiber/siren) adds `actions` — each action specifies a method, an href, a content type, and a list of fields with names and types. It is the `<form>` tag for JSON:

```json
{
  "actions": [
    {
      "name": "add-book",
      "method": "POST",
      "href": "/books",
      "type": "application/json",
      "fields": [
        { "name": "title", "type": "text" },
        { "name": "year", "type": "number" }
      ]
    }
  ]
}
```

A generic Siren client can render that as a form without any prior knowledge of the books API. This is what Fielding means by self-descriptive messages.

[**Collection+JSON**](https://www.iana.org/assignments/media-types/application/vnd.collection+json) defines a `template` property with field definitions for creating and updating collection members.

[**Mason**](https://github.com/JornWildt/Mason) adds `controls` with method, href, schema references, and encoding information.

[**JSON:API**](https://jsonapi.org/) standardizes resource structure and relationship links, though it leans more toward convention than hypermedia discovery.

Each of these is more complete than HAL. Each of these is also less adopted than HAL.

**THE NOVICE asked: "Why?"**

Because HAL is easy. Two keys. No new concepts. Your existing JSON stays exactly the same — you just add `_links` and optionally `_embedded`. A backend developer can adopt HAL in an afternoon. A Siren or Collection+JSON adoption requires rethinking how you model responses, how clients consume them, and possibly firing Kevin, who has strong opinions about response shapes.

Simplicity wins. Even when it wins by being incomplete.

**THE NOVICE said: "This is genuinely sad."**

It is. But it is also the web. The web chose HTML over richer document formats. The web chose HTTP over richer protocols. The web chose `text/plain` over structured alternatives. At every fork, the simpler thing won — not because it was better, but because it was ADOPTABLE. And an adopted half-measure changes the world more than a perfect standard that lives in a specification document that no one reads.

_(Except for Roy Fielding's dissertation, which no one reads but which was STILL RIGHT. Life is not fair.)_

---

## The Fifth Question: Where Does HAL Belong?

**THE NOVICE asked: "Should I use HAL?"**

This depends on what you are using it FOR.

**If you are building a browser application:** use HTML. HTML IS the hypermedia format. It has links AND forms AND controls AND accessibility AND progressive enhancement AND thirty years of browser optimization. Adding HAL to a browser application is like adding a bicycle rack to a car. The car already moves. The bicycle rack is technically functional. Nobody is impressed.

This project — Dothog — uses HTML as its hypermedia format. HTMX extends HTML's capabilities. The `Control` struct maps to HTMX attributes. Forms submit. Links navigate. The browser renders. This is the right architecture for data-centric applications, and no amount of HAL will make it more right.

**If you are building a JSON API that other programs consume:** HAL is a meaningful improvement over bare JSON. Your API consumers discover URLs instead of hardcoding them. Your API can evolve its URL structure without breaking clients. Link relations carry semantic meaning via the IANA registry. This is real value. It is incomplete value — the client still needs out-of-band knowledge for mutations — but it is value.

**If you are building both:** serve HTML to browsers, HAL+JSON to API clients, from the same resource. This is content negotiation. The browser gets `text/html` with full hypermedia controls. The API client gets `application/hal+json` with navigation links. Same resource, same relationships, different media types. The `/hypermedia/hal` demo shows exactly this — the same bookshop resource graph rendered as interactive HTML cards alongside the raw HAL+JSON.

**THE NOVICE said: "So HAL is good enough?"**

HAL is better than nothing. HTML is better than HAL. The question is never "is this format perfect?" The question is "does this format move the client closer to not hardcoding URLs?" And HAL does. Imperfectly. Incompletely. But meaningfully.

There is a MANIFESTO in this project that says: _"There is a school of thought that `application/hal+json` and similar hypermedia JSON formats solve this problem. There is a school of thought that the `<a>` tag solved this problem in 1993. These schools do not talk to each other."_

The HAL demo is these two schools talking to each other. The rendered HTML card and the raw JSON sit side by side. The HTML has buttons that do things. The JSON has links that go places. The gap between those two — the missing `<form>` — is the gap that Fielding would point at. And he would be right. And HAL would still be worth using. And both things can be true.

---

## The Sixth Question: What Does This Teach the Novice?

**THE NOVICE asked: "What should I take from this?"**

Three things.

**First:** The value of HAL is not HAL itself — it is what HAL proves. It proves that ANY format can become hypermedia by embedding links. Two keys. `_links` and `_embedded`. That is all it took to drag JSON from "data dump" to "navigable resource." If JSON can do it, anything can. The bar was never high. The industry just wasn't looking down.

**Second:** The limitation of HAL is the limitation of any format without affordances for unsafe methods. Navigation is discovery — "here is what exists." Affordances are capability — "here is what you can do." A complete hypermedia format needs both. HTML has both. HAL has one. Siren has both. The market chose the one with one. This is not a technical decision. It is a human one.

**Third:** The reason this project includes a HAL demo is not because HAL is the right format for Dothog. It is not. Dothog uses HTML. The reason is that HAL makes the principles _visible in a format developers already understand_. Most developers have never seen `_links` in a JSON response. Most developers have never followed a link relation from one resource to another without constructing a URL. The HAL explorer lets them do this with a click and see both representations — the human one (HTML) and the machine one (JSON) — side by side. And then maybe, just maybe, they look at the HTML side and say: "wait, HTML has been doing this the whole time?"

Yes. Yes it has.

**THE NOVICE closed the document. THE NOVICE opened a browser. THE NOVICE viewed source.**

**It was full of `<a>` tags.**

**THE NOVICE had always known this. THE NOVICE had simply forgotten. The web has a way of making you forget what it is, by showing you so many things built on top of it that you mistake the buildings for the ground.**

**The ground is HTML. The ground has always been HTML.**

---

## Reference

### HAL+JSON Structure

```
Resource
├── _links              (required)
│   ├── self            (conventional, identifies this resource)
│   ├── curies          (optional, compact URI namespaces)
│   └── {rel}           (any IANA or custom link relation)
│       ├── href        (required, the target URL)
│       ├── templated   (optional, true if href is a URI Template)
│       ├── type        (optional, expected media type of target)
│       ├── deprecation (optional, URL of deprecation notice)
│       ├── name        (optional, secondary key for arrays)
│       ├── profile     (optional, additional semantics)
│       └── hreflang    (optional, language of target)
├── _embedded           (optional)
│   └── {rel}           (sub-resources, each a full HAL resource)
└── {properties}        (the resource's own state — plain JSON)
```

### Content Types

| Media Type | Description |
|-----------|-------------|
| `application/hal+json` | HAL in JSON encoding |
| `application/hal+xml` | HAL in XML encoding (less common) |
| `application/prs.hal-forms+json` | [HAL-FORMS](https://rwcbook.github.io/hal-forms/) extension that adds `_templates` with method, fields, and content type — addressing the missing `<form>` |

### Comparison of Hypermedia JSON Formats

| Capability | HAL | Siren | Collection+JSON | Mason | HTML |
|-----------|-----|-------|-----------------|-------|------|
| Navigation links | `_links` | `links` | `links` | namespace links | `<a>` |
| Embedded resources | `_embedded` | `entities` | `items` | — | inline |
| Mutation affordances | — | `actions` | `template` | `controls` | `<form>` |
| Field descriptions | — | `fields` | `data` | `properties` | `<input>` |
| URI templates | `templated: true` | — | — | — | — |
| IANA media type | draft | — | registered | — | registered |
| Adoption | high | low | low | minimal | universal |

### Specifications

- [HAL — draft-kelly-json-hal](https://datatracker.ietf.org/doc/html/draft-kelly-json-hal) — The HAL specification (Internet-Draft, expired but widely implemented)
- [RFC 8288 — Web Linking](https://www.rfc-editor.org/rfc/rfc8288) — Link relation types and the `Link` HTTP header
- [RFC 6570 — URI Template](https://www.rfc-editor.org/rfc/rfc6570) — Template syntax for `templated: true` links
- [IANA Link Relations](https://www.iana.org/assignments/link-relations/link-relations.xhtml) — Registry of standard link relation types
- [HAL-FORMS](https://rwcbook.github.io/hal-forms/) — Extension adding `_templates` for mutation affordances

### In This Project

| Path | What It Does |
|------|-------------|
| `/hypermedia/hal` | Interactive HAL explorer — navigate a bookshop resource graph via HTMX |
| `/hypermedia/hal/api/catalog` | HAL+JSON root resource |
| `/hypermedia/hal/api/books` | HAL+JSON book collection with `_embedded` |
| `/hypermedia/hal/api/books/:id` | HAL+JSON individual book with author link |
| `/hypermedia/hal/api/authors/:id` | HAL+JSON author with book links |
| `/hypermedia/hal/explore?url=...` | HTMX fragment endpoint — renders any HAL resource as an interactive card |

The explorer fetches HAL+JSON from the API endpoints and renders it as interactive HTML alongside the raw JSON. Every `_link` becomes a clickable button that navigates to the target resource. Every `_embedded` resource becomes an expandable card. The dual view makes the relationship between HAL and HTML visible: same data, same relationships, different affordances.
