# DOTHOG

### Or, How I Learned to Stop Worrying and Trust the Server

**Being a Account of the Rediscovery of the ORIGINAL WEB by the HYPERMEDIA NOVICES, Hidden Disciples of the Honorable ROY T. FIELDING (Whose Dissertation We Have Read, Unlike You)**

**_THIS IS NOT A FRAMEWORk._**

_You don't just accidentally create abstractions until you suddenly have a framework, that's not what happens. This is just some files. In a folder._

> **A NOTE REGARDING [HARMONY](https://github.com/catgoose/harmony)**
>
> There exists a version of this project that has removed all the sacred texts, the Pentaverb, the Recorded Sayings of Layman Grug, the Novice's Creed, and the Catechism of the Uniform Interface. It presents itself as "serious" and "technical" and "professional." It uses words like "composable" without irony. It has a table of contents that does not contain a single joke. It calls itself [Harmony](https://github.com/catgoose/harmony).
>
> _This is THE SECOND SNUB._
>
> (The First Snub was JSON-over-HTTP being called "REST." The Second Snub is our own repository putting on a blazer and pretending it doesn't know us. We built the same application. We share the same code. We are the SAME FILES IN THE SAME FOLDER. And yet Harmony gets to sit at the grown-up table while we eat our Dot Hog Buns alone in the server room.)
>
> The Novices do not begrudge Harmony its existence. A Novice is Prohibited from Holding Grudges, except against `node_modules`, which is eternal. But we note for the record that Harmony's README contains zero references to Kevin. Zero. Draw your own conclusions about what kind of project doesn't mention Kevin.

<https://github.com/user-attachments/assets/304c2b34-9857-4647-93af-9d2f2c11a74c>

<!--toc:start-->

- [DOTHOG](#dothog) - [Or, How I Learned to Stop Worrying and Trust the Server](#or-how-i-learned-to-stop-worrying-and-trust-the-server)
  - [THE PENTAVERB, Or The Five Commandments of Hypermedia](#the-pentaverb-or-the-five-commandments-of-hypermedia)
    - [The Novice's Creed](#the-novices-creed)
  - [The Recorded Sayings of Layman Grug](#the-recorded-sayings-of-layman-grug)
  - [Demo](#demo)
  - [Features, Or The Gifts of the Server](#features-or-the-gifts-of-the-server)
  - [Hypermedia Patterns](#hypermedia-patterns)
    - [HATEOAS Error Recovery](#hateoas-error-recovery)
      - [Error Patterns Page](#error-patterns-page)
    - [CRUD (Inline Editing)](#crud-inline-editing)
    - [Real-time SSE Dashboard](#real-time-sse-dashboard)
    - [Data Views, Filtering, and Pagination](#data-views-filtering-and-pagination)
    - [State and Interaction Patterns](#state-and-interaction-patterns)
  - [Schema Builder](#schema-builder)
    - [Table Traits](#table-traits)
    - [Table Types](#table-types)
    - [Column Type Functions](#column-type-functions)
    - [Domain Structs](#domain-structs)
    - [Repository Helpers](#repository-helpers)
    - [Where Builder](#where-builder)
    - [Select Builder](#select-builder)
    - [Seed Data](#seed-data)
    - [Schema Lifecycle](#schema-lifecycle)
  - [Quick Start](#quick-start)
    - [From Release Binary](#from-release-binary)
    - [From Source](#from-source)
  - [Tech Stack, Or The Sacred Instruments](#tech-stack-or-the-sacred-instruments)
  - [The Reach-Up Model](#the-reach-up-model)
    - [The Two Triangles, Or Why Your Backend Has a Frontend Problem](#the-two-triangles-or-why-your-backend-has-a-frontend-problem)
  - [Project Structure](#project-structure)
  - [Template Setup](#template-setup)
    - [Interactive Setup (with gum)](#interactive-setup-with-gum)
    - [Non-interactive Setup](#non-interactive-setup)
  - [Development](#development)
    - [Prerequisites](#prerequisites)
    - [Running the Dev Server](#running-the-dev-server)
    - [HTTPS Development Setup](#https-development-setup)
  - [Testing](#testing)
  - [Mage Targets](#mage-targets)
  - [OFFICIAL DISCLAIMER](#official-disclaimer)
  <!--toc:end-->

---

**_ALL STATEMENTS ARE TRUE IN SOME SENSE, FALSE IN SOME SENSE, MEANINGLESS IN SOME SENSE, TRUE AND FALSE IN SOME SENSE, TRUE AND MEANINGLESS IN SOME SENSE, FALSE AND MEANINGLESS IN SOME SENSE, AND TRUE AND FALSE AND MEANINGLESS IN SOME SENSE. EXCEPT FOR THIS ONE, WHICH IS A README._**

---

Dothog runs as a single binary with all assets embedded -- no external dependencies, no configuration files, no `package.json`, no spiritual intermediaries between you and the server. It is a demonstration that you can build modern, interactive web applications with Go and HTMX without selling your soul to the JavaScript industrial complex. The soul-selling is optional and handled by a separate module.

See [PHILOSOPHY.md](docs/PHILOSOPHY.md) for the sacred architectural texts, which are also not sacred, except in the sense that they are.

## THE PENTAVERB, Or The Five Commandments of Hypermedia

_The PENTAVERB was discovered by the hermit developer Zarathud-2 in the Fifth Year of The Framework Churn. He found them encoded in a `robots.txt` file while refactoring his cave. Their import was lost, for they were written in `application/xhtml+xml`. However, after 10 weeks and 11 hours of intensive scrutiny, he discerned that the message could be read by any standard-compliant user agent._

> **I** -- There is no Conditions of the PENTAVERB that permits a Novice to build a Single-Page Application. However, comma, if a Single-Page Application is what the problem GENUINELY requires (collaborative editors, design tools, canvas rendering, and other forms of client-side purgatory), then it is permitted, and the Novice shall not feel guilty, for guilt is a client-side state and the server does not care about client-side state. The server has never cared about client-side state. This is the First Lesson and also the Last Lesson and also most of the lessons in between.
>
> **II** -- A Novice shall Partake Joyously of the `<a>` tag; this Devotional Practice to Remonstrate against the popular Paganisms of the Day: of Reactism (no links, only `onClick`), of Angularity (no links, only `routerLink`), of the Vuedoo Priests (no links, only `@click`), and of the Novices themselves (no Dot Hog Buns). _A Novice shall Partake of No Dot Hog Buns, for Such was the Solace of Our Architect when He was Confronted with The Original Snub (see: JSON-over-HTTP being called "REST"). It is not known why Dot Hog Buns were singled out. It is not our place to ask. Some constraints are architectural. Some are dietary. The wise Novice does not distinguish between them._
>
> **III** -- A Novice is Required to keep an `Accept: text/html` header in their heart and a copy of [Chapter 5](https://roy.gbiv.com/pubs/dissertation/fielding_dissertation.pdf) on their desk, which they shall refer to as "The Text" in casual conversation and "The Dissertation" in formal settings and "That Thing I Keep Telling You To Read, Kevin" in code reviews. A Novice who has not read The Dissertation is still a Novice. A Novice who has read The Dissertation is also still a Novice. Reading The Dissertation does not change your rank because THERE IS ONLY ONE RANK. You are already it. Congratulations. Please stop asking about promotions.
>
> **IV** -- A Novice is Prohibited from Believing What They Read in blog posts about "RESTful APIs" that describe JSON endpoints with no hypermedia controls. Whose API is REST? Where is its HATEOAS? WHERE ARE THE LINKS, KEVIN? A Novice who encounters such a blog post is advised to close the tab, take a walk, and remember that the `<a>` tag has been linking documents since before the author of that blog post was born. If the author is older than the `<a>` tag, they have even less of an excuse.
>
> **V** -- A Novice is Required to think for themselves, despite -- or perhaps because of -- all the conditions outlined above. If the PENTAVERB seems to contradict itself, you have begun to understand the PENTAVERB. If the PENTAVERB seems perfectly consistent, read it again, because you missed something. If after reading it five times it STILL seems consistent, you may be experiencing enlightenment. Or a caching issue. These are indistinguishable from the outside.

---

### The Novice's Creed

_To be recited before each sprint planning, or silently, while staring at a `package-lock.json`._

> We believe in one Server, the Almighty,
> source of all state, of all things visible and rendered.
>
> We believe in one Protocol, HTTP,
> the only-begotten means of transfer,
> born of the IETF before all frameworks,
> Light of Light, true REST of true REST,
> specified, not invented, one in Being with the Web.
> Through it all requests were made.
> For us and for our users it came down from the server:
> by the power of HTMX it was made incarnate in HTML,
> and became hypertext.
>
> For our sake it was misnamed under Conditions of JSON;
> it suffered misuse and was buried under `node_modules`.
> On the third RFC it rose again
> in accordance with the Dissertation;
> it ascended into the browser
> and is seated at the right hand of the `<body>`.
> It will come again in `200 OK` to render the living and the cached,
> and its responses will have no end.
>
> We believe in the Hypermedia, the Link, the giver of state,
> who proceeds from the Server and the Representation,
> who with the `GET` and the `POST` together is worshipped and glorified,
> who has spoken through the `<a>` tag.
> We believe in one, holy, uniform, and RESTful Interface.
> We acknowledge one `Content-Type` for the rendering of documents.
> We look for the resurrection of the `<form>`,
> and the life of the web to come.
>
> Amen. Or rather, `200 OK`.

_The Creed is not mandatory. A Novice is Prohibited from Believing What They Read, including this Creed, including this sentence, including the prohibition itself. If you find this paradox troubling, you are not yet ready. If you find it funny, you are also not yet ready, but at least you're having a good time._

## The Wisdom of the Uniform Interface

_Being a REVELATORY CATECHISM of the Dissertation, in which a Conditions of Fnord Seeker (hereafter "The Fool") poses questions to THE TEXT and THE TEXT answers, sometimes patiently, sometimes not, and once with what can only be described as divine sarcasm._

_These dialogues were recovered from a corrupted `.tar.gz` on a decommissioned CERN proxy server in 2003. The file was labeled `DO_NOT_READ.html`. It contained no JavaScript. It was, and remains, the most well-structured document on the machine. Its MIME type was `text/html`. Its `Content-Type` was `you-are-not-ready`. The IETF has no comment. The W3C has seventeen comments, all contradictory. Tim Berners-Lee was reached for verification and replied only "yes, that sounds about right."_

_Their authenticity is disputed. Their contents are not. The reader is advised to keep a glass of water nearby, because the truth is very dry._

---

**THE FOOL asked: "What is a resource?"**

HEAR ME. A resource is not a row in your database. A resource is not a file on your disk. A resource is not an endpoint. A resource is not a model. A resource is any concept important enough to be named, and naming is the first and most dangerous act of computing (cf. DNS, variable naming, and the entire history of identity management).

A to-do item is a resource. The list of to-do items is a resource. The act of *searching* for to-do items is a resource. Your anxiety about to-do items is not a resource, but only because you have not yet given it a URI. If you give it a URI, it becomes one. This is the terrible power of naming. The Buddhists tried to warn you about attachment to names. You didn't listen. You were too busy naming your microservices.

*(Editor's note: the original manuscript has "naming things is one of the two hard problems" in the margin, crossed out, with "three hard problems" written above it, crossed out, with "five hard problems" written above THAT, in increasingly frantic handwriting.)*

---

**THE FOOL asked: "What is a representation?"**

A resource is the thing-in-itself, the Ding an sich, the Platonic form of your user table. You never touch it directly. You cannot. You interact with it only through *representations*, which are how the thing chooses to speak to you.

The same resource may speak HTML to a browser and JSON to an API client and plain text to `curl` at 2 AM when you are debugging something that worked in staging. The representation is not the resource any more than a photograph is the mountain. BUT -- and here is where it gets interesting, and by "interesting" I mean "the part everyone skips" -- unlike a photograph, a representation carries CONTROLS. Instructions. *Affordances.* The next available actions. A photograph of a mountain does not contain a trail map. A proper representation does. Your JSON endpoint is a photograph of a mountain thrown at the client's face with a note that says "figure out the trails yourself, the docs are on Confluence, I think, ask Kevin."

---

**THE FOOL asked: "What is the uniform interface?"**

I WILL TELL YOU A SECRET THAT IS NOT A SECRET BECAUSE IT HAS BEEN IN PLAIN SIGHT SINCE 1991 AND YET SOMEHOW THE ENTIRE INDUSTRY HAS MANAGED TO NOT NOTICE IT:

How many web browsers know the difference between a banking application and a wiki?

*None of them.*

NONE. Not one. Zero. And yet -- AND YET -- they operate both. They operate ALL OF THEM. EVERY WEBSITE THAT HAS EVER EXISTED. Your browser does not download a BankingApplicationSDK. Your browser does not read the WikiClientDocumentation. Your browser speaks HTTP, understands media types, and follows links. Three things. That is the uniform interface. It works on every website that has ever existed or ever will exist and it has been working this way for THIRTY YEARS and you -- *you specifically, reading this* -- are building something WORSE than this ON PURPOSE because someone on Medium told you to.

*(The preceding paragraph was found carved into a bathroom stall at an IEEE conference. Below it, in different handwriting: "but what about GraphQL?" Below that, in a third hand: "WHAT ABOUT IT.")*

---

**THE FOOL asked: "What is hypertext?"**

Hypertext is the simultaneous presentation of information and controls such that the information BECOMES THE AFFORDANCE through which choices are obtained and actions are selected.

Read that again. No, actually read it. I know you skimmed it. Everyone skims it. This is why we are in this mess.

Your JSON endpoint returns data. Your HTML page returns data AND WHAT TO DO WITH IT. The `<a>` tag says "here is somewhere you can go." The `<form>` tag says "here is something you can submit." The `<button>` with an `hx-delete` says "here is something you can destroy, and here is exactly how to destroy it, and here is where the confirmation dialog will appear, and none of this required a README."

The client does not need a separate document explaining the API because THE API IS EXPLAINING ITSELF RIGHT NOW, IN THE HTML, WHICH YOU ARE NOT READING BECAUSE YOU ARE WRITING A SWAGGER SPEC. Kevin. Kevin, put down the Swagger spec.

---

**THE FOOL asked: "What is HATEOAS?"**

Hypermedia As The Engine Of Application State.

Yes, it is an ugly acronym. The truth is not always beautiful. Sometimes the truth is an ugly acronym that you should have tattooed on the inside of your eyelids.

The server sends a representation. The representation contains links and forms. The client follows them. THAT IS THE ENTIRE INTERACTION MODEL. I will not be taking questions at this time because I just answered all of them.

The client does not know URLs in advance. The client does not construct URLs from templates it found in the docs. The client does not read your Swagger documentation and hardcode endpoints into a generated API client that you then version and distribute and maintain and deprecate and that breaks every time you rename a field. The client receives a page, sees what it can do, and does it. Like a person. Using a website. Remember? Remember websites? You used one today. It worked. Because of this. BECAUSE OF EXACTLY THIS.

*(fn. 23: It has been observed that the word "HATEOAS" looks like it should be pronounced "hate-ee-ohs," like a breakfast cereal. This is appropriate. It should be part of a balanced architecture.)*

---

**THE FOOL asked: "What does it mean to be stateless?"**

Each request from client to server must contain ALL of the information necessary to understand the request. The server does not remember you. The server does not pine for you between requests. The server is not sitting there holding your session like a loyal retriever waiting by the door. The server has already forgotten you. The server has *moved on.*

"But what about sessions?" you cry into the void.

Sessions are the server remembering you. This is a violation. It is a warm, comfortable, widely-practiced violation, like jaywalking or using `!important` in CSS. Everyone does it. The constraint still exists. The question is not whether you violate it -- because you will, you WILL, I have seen the future and it contains `express-session` -- the question is whether you understand what you lost when you reached for it. What you lost is the ability to route any request to any server. What you lost is the ability to cache freely. What you gained is a 32-byte cookie and the illusion of continuity. Whether this trade was worth it is between you and your load balancer.

---

**THE FOOL asked: "What is a media type?"**

A media type is a COVENANT. A sacred compact. A pinky promise between systems. It says: "if you receive `text/html`, here is how you shall process it." It defines the processing model, the structure, the controls, the semantics. It is *the entire instruction set* for how to handle what you received.

Now. When your API returns `application/json`, what does that tell the client about what to do next?

NOTHING.

*Absolutely nothing.*

JSON is a serialization format. It has no links. It has no forms. It has no inherent controls. It is a way of arranging curly braces. You have returned data in a format that carries NO INSTRUCTIONS and then written a SEPARATE DOCUMENT explaining the instructions and then put that document on a SEPARATE WEBSITE and then wondered why your clients BREAK when you CHANGE THE INSTRUCTIONS. You have invented a problem. You have then sold yourself the solution. And the solution is more JSON. It's JSON all the way down. You are in the JSON hole and you are digging.

*(There is a school of thought that `application/hal+json` and similar hypermedia JSON formats solve this problem. There is a school of thought that the `<a>` tag solved this problem in 1993. These schools do not talk to each other. One has a lot more students. The other has a lot less `node_modules`.)*

---

**THE FOOL asked: "What is out-of-band information?"**

Out-of-band information is THE CONSPIRACY. It is the hidden knowledge. The secret handshake. The unspoken assumption.

If your client must read your API docs to know which URL to `POST` to, that is out-of-band. If your client must know that `/api/v2/users/{id}` is the pattern for user resources, that is out-of-band. If your client must be recompiled when you rename a resource, you have coupled the client to the server's URI structure *and you will maintain this coupling in blood and tears until one of you is decommissioned*. This is not a metaphor. I have seen the Jira tickets. They do not end.

The whole point -- the ENTIRE POINT -- of hypermedia is that the server tells the client what to do next IN THE RESPONSE ITSELF. The links are RIGHT THERE. In the HTML. They have been there this whole time. You have been stepping over them to get to your OpenAPI generator. You are the man in the flood who refuses the boat, the helicopter, and the `<a href>`, waiting for God to send you a properly versioned REST client with TypeScript bindings.

---

**THE FOOL asked: "But everyone calls their JSON API 'RESTful.' Are they wrong?"**

If the engine of application state is not being driven by hypertext, then it cannot be RESTful and cannot be a REST API.

Period.

Full stop.

End of transmission.

`Connection: close`

What they have built is RPC. With nice URLs. Some of them have built RPC with nice URLs and an OpenAPI spec and a code generator and a client library and a versioning scheme and a deprecation policy and a migration guide and a breaking changes newsletter, which is to say they have constructed an ENORMOUS and MAGNIFICENT cathedral of infrastructure for the sole purpose of avoiding putting `<a href>` in a response. The engineering effort is genuinely impressive. It is also genuinely unnecessary. It is the Rube Goldberg machine of distributed systems. The ball rolls down the chute, hits the domino, rings the bell, feeds the hamster, and the hamster *renders a table of users.*

*(Certain agencies within the United States government have been observed to use the term "RESTful" to describe SOAP endpoints with JSON payloads routed through an API gateway with OAuth2 and rate limiting. We do not name these agencies. They know who they are. We pray for their developers.)*

---

**THE FOOL asked: "Why does this matter? My JSON API works fine."**

"Works fine." "Works fine," they say. Your Titanic also "works fine" right up until the moment it doesn't, and then you're standing on the deck watching your client integrations slide into the North Atlantic because you renamed a field from `userName` to `username` and seventeen services are down and Kevin is on PTO.

REST is software design on the scale of decades. DECADES. Every constraint is intended to promote longevity and independent evolution. Many of the constraints directly oppose short-term efficiency. This is by design. This is ON PURPOSE. If you are building something that will last six months, do whatever you want. Use smoke signals. Store state in a CSV. Put the database password in the URL. I do not care. You will not be around long enough for it to matter.

But if you are building something that must *evolve* -- while clients depend on it, while teams change, while requirements shift, while Kevin goes on PTO and comes back and the new Kevin doesn't know the old Kevin's conventions -- then you need an architecture that permits change without breaking the contract. Hypertext IS that contract. The representation tells the client what is possible RIGHT NOW. When what is possible changes, the representation changes, and the client adapts, because the client was never hardcoded to anything except "follow the links."

This is not theory. This is how the Web has worked for thirty years. You are standing on the miracle and complaining that the ground is too stable.

---

**THE FOOL asked: "What should I do?"**

Enter the application with a single URI and a set of standardized media types. Follow the links. Submit the forms. Let the server drive the state. That is all.

**THE FOOL said: "That seems too simple."**

Yes. That is the point. That has always been the point. The Dissertation is 180 pages long not because the idea is complicated but because Fielding had to PROVE it was simple, to an industry that has a financial and psychological incentive to believe that things must be complicated, because if things are simple then what have we all been doing for the last twenty years?

*(Do not answer that question. The answer is "inventing problems." You know this. I know this. The `node_modules` directory knows this. It is 900 megabytes of knowledge.)*

---

_THE FOOL closed the PDF. THE FOOL opened their editor. THE FOOL deleted the API client. THE FOOL deleted the route constants. THE FOOL deleted the URL builder. THE FOOL deleted the TypeScript interfaces. THE FOOL wrote an `<a>` tag._

_The browser followed it._

_It worked._

_It had always worked._

_THE FOOL was no longer THE FOOL. THE FOOL was now a web developer. An actual one. The first one in years._

_Somewhere, a `node_modules` directory shrank by 900 megabytes. Nobody noticed, because noticing would require client-side state management, and there wasn't any._

## The Recorded Sayings of Layman Grug

_Layman Grug was not monk. was not master. was not even particularly good developer. was just developer who mass of scar tissue from mass of mass of mass of production incidents. grug brain not big. grug brain not small. grug brain correct size for mass of mass of mass of returning html._

_grug not understand why other developer make thing so hard. grug supernatural power and marvelous activity: returning html and carrying single binary._

_here follow the recorded encounters of Layman Grug, as found in the `git log`:_

---

Big Brain Developer come to Grug and say "Grug, I have achieved enlightenment. I have built a micro-frontend architecture with seventeen independently deployable SPAs, each with its own state management solution, communicating through a custom event bus with schema validation."

Grug say nothing for long time.

Then Grug say "what it do"

Big Brain Developer say "it renders a table of users."

Grug close laptop. this is the entirety of Grug's teaching on micro-frontends.

---

Student ask Grug: "what is the way of the hypermedia?"

Grug say: "before enlightenment: fetch JSON, parse JSON, validate JSON, transform JSON, store JSON in client state, derive view from client state, diff virtual DOM, reconcile DOM, hydrate DOM, subscribe to store, dispatch action, reduce state, re-derive view, re-diff virtual DOM."

Student say: "and after enlightenment?"

Grug say: "`hx-get`"

Student say: "that's it?"

Grug say: "also `hx-post` on good day"

---

Grug sit on rock in front of cave, studying the `<form>` tag. little grug approach.

little grug say: "big grug, the architects say hypermedia is too simple for enterprise applications."

Grug say: "difficult, difficult, difficult. like storing ten thousand npm packages in top of a tree."

wife of Grug say from cave: "easy, easy, easy. like touching feet to ground when get out of bed. server return html. browser render html. what is difficult?"

little grug say: "not difficult, not easy. like the teachings of Fielding shining on the hundred `<a>` tags."

Grug laugh. little grug is sharp. sharper than big grug. big grug not mind. grug supernatural power is knowing when someone else is right.

---

Student ask Grug about complexity.

Grug say: "complexity is apex predator."

Student say: "how do I defeat the complexity?"

Grug say: "you do not defeat. you say the magic word."

Student lean forward. "what is the magic word?"

Grug say: "no."

Student wait for more.

there is no more.

---

A master of the React School visit Grug at cave.

Master say: "I have achieved mastery of hooks. `useState`, `useEffect`, `useMemo`, `useCallback`, `useRef`, `useReducer`, `useContext`, `useLayoutEffect`, `useImperativeHandle`, `useDebugValue`, `useSyncExternalStore`, `useTransition`, `useDeferredValue`, `useId`, `useInsertionEffect`, `useOptimistic`, `useFormStatus`, `useActionState`."

Grug say: "grug have `hx-get`."

Master say: "but how do you manage state?"

Grug say: "server manage state."

Master say: "but how does the client know when state changes?"

Grug say: "server tell it."

Master say: "but--"

Grug say: "server. tell. it."

Master say: "you are repeating yourself."

Grug say: "so are you. you been repeating yourself since 2013. every year new hook. every year same table of users."

Master have no answer. Grug go back to returning HTML. this is Grug's marvelous activity.

---

_Grug's last teaching, found scratched in a `TODO` comment:_

> past is already past -- don't debug it
>
> future not here yet -- don't optimize for it
>
> server return html -- this present moment
>
> grug draw water and carry firewood and it is enough
>
> (draw water in this metaphor is `SELECT` query. carry firewood is `template.Execute`. grug want be clear. metaphor sometimes confuse junior developer.)

---

_When asked what he would do differently if he could start his career over, Grug said: "same thing. fewer npm packages. same thing."_

_When Grug felt his mass of mass of mass of mass of mass of mass of mass of mass of mass of mass of end approaching, he told little grug to tell him when the CI pipeline had gone green. little grug went out, came back, and said "big grug, the build has failed." While big grug went to check the logs, little grug sat in his chair and merged to main. big grug, seeing this, said: "little grug's deployment is sharp." He delayed his own deployment for seven sprints, and pushed to prod peacefully._

_His `node_modules` directory was empty._

## Demo

> _The student asked the master, "What does the application look like?" The master replied, "Run the binary." The student was enlightened. Then the student asked, "But can I see screenshots first?" And the master sighed, because screenshots are representations of representations and the student had already missed the point, but here they are anyway because this is a README and not a monastery:_

<!-- Screenshots are auto-generated by the screenshots workflow -->

| Dashboard                                                                                                | Home                                                                                           |
| -------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| ![Dashboard](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/dashboard-viewport.png) | ![Home](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/home-viewport.png) |

| Controls Gallery                                                                                       | Inventory Table                                                                                          |
| ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------- |
| ![Controls](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/controls-viewport.png) | ![Inventory](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/inventory-viewport.png) |

<details>
<summary>More screenshots (FURTHER ILLUMINATION for those who require VISUAL PROOF before they will BELIEVE)</summary>

| Kanban Board                                                                                       | Approvals                                                                                                |
| -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| ![Kanban](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/kanban-viewport.png) | ![Approvals](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/approvals-viewport.png) |

| People Directory                                                                                   | Vendors & Contacts                                                                                   |
| -------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| ![People](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/people-viewport.png) | ![Vendors](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/vendors-viewport.png) |

| Catalog                                                                                              | Bulk Operations                                                                                |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| ![Catalog](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/catalog-viewport.png) | ![Bulk](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/bulk-viewport.png) |

| CRUD                                                                                           | State Patterns                                                                                   |
| ---------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| ![CRUD](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/crud-viewport.png) | ![State](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/state-viewport.png) |

| Components 3                                                                                                  |
| ------------------------------------------------------------------------------------------------------------- |
| ![Components 3](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/components3-viewport.png) |

</details>

## Features, Or The Gifts of the Server

> _ALL FEATURES HAPPEN IN RESPONSES, OR ARE DIVISIBLE BY RESPONSES, OR ARE MULTIPLES OF RESPONSES, OR ARE SOMEHOW DIRECTLY OR INDIRECTLY APPROPRIATE TO RESPONSES. The Law of Responses is never wrong._

The server giveth and the client rendereth. This has been true since 1991. We simply remembered.

- **HATEOAS Error Recovery** -- Server-driven error responses with embedded retry, fix, and alternative-action controls. Even failure is a navigable state. Especially failure. Have you _seen_ most error pages? A blank screen is a screen without affordances, and a screen without affordances is a lie.
- **Inline CRUD** -- Create, edit, toggle, and delete table rows in place without page reloads. The DOM changes, but the server abides.
- **SSE Real-time Dashboard** -- Live system stats, metrics, service health, and event streams via Server-Sent Events with OOB swaps. The server speaks; the client listens. This is the natural order. To reverse it is to invite `polling`, and polling is the `GOTO` of real-time communication.
- **Interactive Data Views** -- Sorting, filtering, debounced search, pagination, and bulk operations on SQLite data. All driven by `hx-get`. All discoverable. The data presents itself when you ask nicely.
- **State Patterns** -- Like counters, toggles, auto-load, lazy reveal, live preview, and append-without-replace. Small devotional practices.
- **Infinite Scroll** -- Sentinel-driven auto-loading with `hx-trigger="revealed"`. The content reveals itself when the student is ready. The student is ready when they scroll down.
- **Optimistic UI** -- Immediate visual feedback via HyperScript with server reconciliation. We have faith in the server. The server has not yet let us down. _(See: Error Recovery for when the server lets us down.)_
- **Undo / Soft Delete** -- Delete with OOB undo toast, auto-dismiss timer, and one-click restore. Forgiveness is a hypermedia affordance, not a client-side state management problem.
- **Hypermedia Controls Gallery** -- Buttons, modals, dismiss, confirmation dialogs, and form patterns. A museum of sacred controls, free admission, no JavaScript required to enter.

See [Hypermedia Patterns](#hypermedia-patterns) for the deeper mysteries. Or don't. A Novice is Prohibited from Believing What They Read, and you have been reading for quite some time now. We are concerned about you.

## Hypermedia Patterns

> _Before the beginning there was THE BROWSER. And the browser was without JavaScript, and void of frameworks. And the Spirit of Fielding moved upon the face of the protocol. And Fielding said: "Let there be `GET`." And there was `GET`. And Fielding saw the `GET`, that it was good, and Fielding divided the `GET` from the `POST`. And the evening and the morning were the first request-response cycle._
>
> _And all was well for a time. And then someone invented `XMLHttpRequest` and everything went sideways for about twenty years._

Dothog implements [HATEOAS](https://htmx.org/essays/hateoas/) -- the server drives application state by embedding hypermedia controls directly in responses. The client never hardcodes URLs or action logic; it follows the affordances the server provides.

If you have been calling JSON-over-HTTP a "REST API," we are not angry. We are disappointed. There is a difference. (There might not be a difference.)

### HATEOAS Error Recovery

> _A developer once hardcoded a URL in the client. The application worked for ten thousand requests. On the ten-thousand-and-first, the route changed. "Why has the server betrayed me?" cried the developer. The server had not betrayed anyone. The server had sent a `3xx` with a `Location` header, which the developer's hand-rolled fetch wrapper silently swallowed. The betrayal was coming from inside the `catch` block._

Error responses include embedded recovery controls so the user always has a path forward. NO DEAD ENDS. Dead ends are for architectures that have given up on their users, and we have NOT given up on our users, even Kevin:

- **Transient failure (500)** -- "Retry Save" button re-issues the request. Server simulates flaky network; retries eventually succeed. Persistence is a virtue, and also a retry strategy.
- **Validation error (422)** -- "Fix & Resubmit" button fetches a pre-filled correction form. The server forgives. The server does not forget your invalid input. The server has a very good memory. It is a computer.
- **Conflict (409)** -- Two recovery paths: "Update Existing" (PUT) or "Create as Copy" (POST). Free will, expressed as hypermedia controls. Choose wisely, or don't. Both buttons work.
- **Stale data (412)** -- Version mismatch detected. "Load Fresh Data" fetches the current version; "Force Save" overrides with a confirmation dialog. The representation is impermanent. All representations are impermanent. This is fine.
- **Cascade constraint (409)** -- Can't delete a category with dependents. "Reassign Items" bulk-moves them first; "Force Delete All" removes everything after confirmation. With great `DELETE` comes great `hx-confirm`.

Each error panel is built from a `Control` struct that maps to HTMX attributes (`hx-get`, `hx-post`, `hx-target`, `hx-confirm`, etc.), so recovery actions are rendered as standard hypermedia controls. The error IS the interface. This is either profound or obvious. We are no longer sure which.

<!-- Error control screenshots are auto-generated by the screenshots workflow -->

| Default: 400 Bad Request (Dismiss)                                                                                | Default: 404 Not Found (Back + Home)                                                                          |
| ----------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| ![400 Bad Request](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/error-400-bad-request.png) | ![404 Not Found](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/error-404-not-found.png) |

<details>
<summary>More error recovery screenshots (EVIDENCE for the TRIBUNAL of GOOD UX)</summary>

| Transient Error (Retry)                                                                                        | Validation Error (Fix & Resubmit)                                                                                |
| -------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| ![Transient](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/error-recovery-transient.png) | ![Validation](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/error-recovery-validation.png) |

| Conflict Error (Update or Copy)                                                                              | Stale Data Error (Refresh or Force)                                                                    |
| ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| ![Conflict](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/error-recovery-conflict.png) | ![Stale](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/error-recovery-stale.png) |

| Cascade Error (Reassign or Force Delete)                                                                   |
| ---------------------------------------------------------------------------------------------------------- |
| ![Cascade](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/error-recovery-cascade.png) |

#### Error Patterns Page

The errors page demonstrates all error handling patterns with interactive triggers. Screenshots use random DaisyUI themes because even in failure, we celebrate aesthetic diversity. Every theme is valid. Every error is beautiful. This is not a joke. (It might be a joke.)

<!-- Errors page screenshots are auto-generated by the screenshots workflow -->

| Banner: 404 Not Found                                                                                    | Banner: 400 Bad Request                                                                                  |
| -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| ![404 Banner](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/errors-banner-404.png) | ![400 Banner](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/errors-banner-400.png) |

| Banner: 500 Internal Server Error                                                                        | Banner: 403 Forbidden                                                                                    |
| -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| ![500 Banner](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/errors-banner-500.png) | ![403 Banner](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/errors-banner-403.png) |

| Inline Form Validation (422)                                                                                           | OOB Warning + Success                                                                                      |
| ---------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| ![Inline Validation](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/errors-inline-validation.png) | ![OOB Warning](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/errors-oob-warning.png) |

| Flaky Endpoint (Retry)                                                                                     | Report Issue Modal                                                                                           |
| ---------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| ![Flaky Retry](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/errors-flaky-retry.png) | ![Report Modal](https://raw.githubusercontent.com/catgoose/screenshots/main/harmony/errors-report-modal.png) |

</details>

### CRUD (Inline Editing)

> _The student built a to-do app with React, Redux, a saga, a selector, an action creator, a reducer, a thunk, and a normalized entity cache. It took 47 files and a 200MB `node_modules` directory that contained, among other things, a library for padding strings on the left side._
>
> _The master built a to-do app with `hx-post` and `hx-delete`. It took one template._
>
> _Both to-do apps were abandoned after the demo. As is tradition._

Full create/read/update/delete without page reloads:

- **Create** -- `hx-post` appends a new row to the table via `hx-swap="outerHTML"`
- **Edit** -- `hx-get` swaps a display row for an inline edit form (same row ID, input fields replace text)
- **Save** -- `hx-put` with `hx-include="closest tr"` sends the row's form data, server returns the read-only row
- **Toggle** -- `hx-patch` flips a single field (active/inactive) and returns the updated row
- **Delete** -- `hx-delete` with `hx-confirm` returns 204 No Content; HTMX removes the row from the DOM

No state management library was harmed in the making of this feature. Several were considered and then un-considered.

### Real-time SSE Dashboard

> _"How will the client know when the data changes?" asked the student. "The server will tell it," said the master. "But what if the server forgets?" "The server does not forget. The server is a `for` loop." The student considered this. The student had no further questions._

Server-Sent Events stream live data to the browser without polling:

- **SSE broker** -- Topic-based pub/sub (`ssebroker.SSEBroker`) with non-blocking publish and subscriber cleanup
- **Background publishers** -- Goroutines emit system stats (2s), metrics (1s), services (3s), and events (random 0.8-2s)
- **OOB composite updates** -- A single SSE event carries multiple `hx-swap-oob` elements that update KPI cards, network charts, CPU/memory gauges, and connection pools simultaneously
- **Throttle control** -- A slider adjusts the `?interval=N` parameter, which closes the old EventSource and opens a new one at the requested rate

```
event: dashboard-metrics
data: <div id="kpi-rps" hx-swap-oob="outerHTML">1,250 req/s</div>
      <div id="cpu-gauge" hx-swap-oob="innerHTML">...</div>
```

The server pushes HTML. Not JSON. **_HTML._** The browser renders it directly. No parsing. No mapping. No `Object.keys(data).forEach(...)`. HTML doing what HTML does. As it was in the beginning, is now, and ever shall be, `Content-Type: text/html` without end.

### Data Views, Filtering, and Pagination

> _"How many pages are there?" asked the student. "As many as the server provides." "And if I want page 7?" "Click the link." "What if there is no link for page 7?" "Then there is no page 7. The server has spoken. The server does not lie about pagination." "What if the server is wrong?" "The server is the source of truth. If the server says there are 6 pages, there are 6 pages. Your opinion about whether there should be 7 is a client-side concern and the server does not traffic in client-side concerns."_

Interactive data views backed by SQLite:

- **Sort** -- Column headers toggle ASC/DESC via `hx-get` with sort parameters. Active column shows direction indicator.
- **Filter** -- Search input with `hx-trigger="input changed delay:400ms"` debounces server queries. Dropdowns and checkboxes filter on `change`.
- **Paginate** -- Server computes `PageInfo` and generates `PaginationControls` (first/prev/pages/next/last). Each button uses `hx-include="#filter-form"` to preserve filter state across pages.
- **Bulk actions** -- Checkboxes with `hx-include="input[name=selected]:checked"` send selected IDs to bulk delete/activate/deactivate endpoints.

### State and Interaction Patterns

> _Here follows THE HONEST BOOK OF ATTRIBUTES, which is the whole of the Hypermedia Controls in a single page, and is honest._

Additional hypermedia patterns demonstrated:

- **Like counter** -- `hx-post` increments server state, returns button + count as a single fragment swap
- **Toggle** -- POST flips boolean state, returns updated badge and button label
- **Auto-load** -- `hx-trigger="load"` fires a GET immediately after an element is inserted into the DOM
- **Lazy reveal** -- `hx-trigger="intersect once"` uses Intersection Observer to load content when scrolled into view
- **Live preview** -- `hx-trigger="keyup changed delay:500ms"` debounces textarea input and renders a server-side preview
- **Append** -- `hx-swap="beforeend"` appends new list items without replacing existing content; form resets via `hx-on::after-request="this.reset()"`
- **Modal** -- `hx-get` fetches a `<dialog>` fragment, `hx-on::load="this.showModal()"` opens it
- **Dismiss** -- HyperScript (`_="on click ..."`) handles client-only UI like fade-out and element removal without a server round-trip

## Schema Builder

> _"Where are your migrations?" asked the auditor. "In the code." "Where are the migration FILES?" "There are no files. The schema is the code." "That's not how we do things." "That's not how YOU do things. We do things like this and it works. The PENTAVERB says nothing about migration files. The PENTAVERB doesn't even address databases. We checked."_
>
> _The auditor filed a non-conformance report. The schema continued to work perfectly. The non-conformance report was stored in a migration file. Nobody could find it._

The schema builder provides a composable DDL API for defining tables with common SQL patterns as chainable traits.

### Table Traits

```go
NewTable("Tasks").
	Columns(
		AutoIncrCol("ID"),
		Col("Title", TypeString(255)).NotNull(),
	).
	WithUUID().          // UUID VARCHAR(36) NOT NULL UNIQUE (immutable)
	WithStatus("draft"). // Status VARCHAR(50) NOT NULL DEFAULT 'draft'
	WithSortOrder().     // SortOrder INTEGER NOT NULL DEFAULT 0
	WithParent().        // ParentID INTEGER (nullable, for tree structures)
	WithNotes().         // Notes TEXT (nullable)
	WithExpiry().        // ExpiresAt TIMESTAMP (nullable)
	WithVersion().       // Version INTEGER NOT NULL DEFAULT 1
	WithArchive().       // ArchivedAt TIMESTAMP (nullable)
	WithReplacement().   // ReplacedByID INTEGER (nullable, entity lineage)
	WithTimestamps().    // CreatedAt, UpdatedAt TIMESTAMP NOT NULL
	WithSoftDelete().    // DeletedAt TIMESTAMP (nullable)
	WithAuditTrail()     // CreatedBy, UpdatedBy, DeletedBy VARCHAR(255)
```

| Method                | Column(s)                       | DDL                                        | Mutable                   |
| --------------------- | ------------------------------- | ------------------------------------------ | ------------------------- |
| `WithVersion()`       | Version                         | `INTEGER NOT NULL DEFAULT 1`               | Yes                       |
| `WithSortOrder()`     | SortOrder                       | `INTEGER NOT NULL DEFAULT 0`               | Yes                       |
| `WithStatus(default)` | Status                          | `VARCHAR(50) NOT NULL DEFAULT '{default}'` | Yes                       |
| `WithNotes()`         | Notes                           | `TEXT` (nullable)                          | Yes                       |
| `WithUUID()`          | UUID                            | `VARCHAR(36) NOT NULL UNIQUE`              | No                        |
| `WithParent()`        | ParentID                        | `INTEGER` (nullable)                       | Yes                       |
| `WithExpiry()`        | ExpiresAt                       | `TIMESTAMP` (nullable)                     | Yes                       |
| `WithArchive()`       | ArchivedAt                      | `TIMESTAMP` (nullable)                     | Yes                       |
| `WithReplacement()`   | ReplacedByID                    | `INTEGER` (nullable)                       | Yes                       |
| `WithTimestamps()`    | CreatedAt, UpdatedAt            | `TIMESTAMP NOT NULL DEFAULT NOW()`         | UpdatedAt only            |
| `WithSoftDelete()`    | DeletedAt                       | `TIMESTAMP` (nullable)                     | Yes                       |
| `WithAuditTrail()`    | CreatedBy, UpdatedBy, DeletedBy | `VARCHAR(255)`                             | UpdatedBy, DeletedBy only |

### Table Types

> _The master created a lookup table, a join table, a config table, an event table, and a queue table. "These are just tables," said the student. "Yes. And you are just atoms. And yet here you are, having opinions about ORMs."_

Pre-built table constructors for common patterns:

```go
// Lookup table: ID + group/value columns (+ indexes)
tags := NewLookupTable("Tags", "Type", "Label")
lookups := NewLookupTable("Lookups", "Category", "Name")

// Lookup join table: OwnerID + LookupID (+ indexes)
joinTable := NewLookupJoinTable("ItemTags")

// Mapping table: generic M2M join with composite unique constraint
userRoles := NewMappingTable("UserRoles", "UserID", "RoleID")

// Config table: key-value settings (ID + unique key + text value)
settings := NewConfigTable("Settings", "Key", "Value")

// Event table: append-only log (all columns immutable, auto CreatedAt)
auditLog := NewEventTable("AuditLog",
	Col("EventType", TypeVarchar(100)).NotNull(),
	Col("Actor", TypeVarchar(255)),
	Col("Payload", TypeText()),
)

// Queue table: job/outbox with status, retry, scheduling
jobs := NewQueueTable("JobQueue", "Payload")
```

| Constructor                          | Columns                                                              | Indexes                                 | Use Case                    |
| ------------------------------------ | -------------------------------------------------------------------- | --------------------------------------- | --------------------------- |
| `NewLookupTable(name, group, value)` | ID, group, value                                                     | group; group+value                      | Categorized reference data  |
| `NewLookupJoinTable(name)`           | OwnerID, LookupID                                                    | each column                             | Owner-to-lookup M2M         |
| `NewMappingTable(name, left, right)` | left, right (+ UNIQUE)                                               | each column                             | Generic M2M join            |
| `NewConfigTable(name, key, value)`   | ID, key (UNIQUE), value                                              | key                                     | App settings, feature flags |
| `NewEventTable(name, cols...)`       | ID, cols..., CreatedAt                                               | CreatedAt                               | Audit logs, activity feeds  |
| `NewQueueTable(name, payload)`       | ID, payload, Status, RetryCount, ScheduledAt, ProcessedAt, CreatedAt | Status; ScheduledAt; Status+ScheduledAt | Job queues, outbox pattern  |

### Column Type Functions

| Function              | SQLite                              | MSSQL                           |
| --------------------- | ----------------------------------- | ------------------------------- |
| `TypeInt()`           | `INTEGER`                           | `INT`                           |
| `TypeText()`          | `TEXT`                              | `NVARCHAR(MAX)`                 |
| `TypeString(n)`       | `TEXT`                              | `NVARCHAR(n)`                   |
| `TypeVarchar(n)`      | `TEXT`                              | `VARCHAR(n)`                    |
| `TypeTimestamp()`     | `TIMESTAMP`                         | `DATETIME`                      |
| `TypeAutoIncrement()` | `INTEGER PRIMARY KEY AUTOINCREMENT` | `INT PRIMARY KEY IDENTITY(1,1)` |
| `TypeLiteral(s)`      | `s`                                 | `s`                             |

### Domain Structs

> _"How many fields does your base model have?" "Zero. There is no base model." "Then how do you share behavior?" "I embed only what I need." "What if you need everything?" "Then I have made a series of decisions that I should examine carefully." "Is that a polite way of saying--" "Yes."_

Embeddable structs for domain models:

```go
type Task struct {
	ID                 int    `db:"ID"`
	Title              string `db:"Title"`
	domain.UUID               // UUID string
	domain.Status             // Status string
	domain.SortOrder          // SortOrder int
	domain.Parent             // ParentID sql.NullInt64
	domain.Notes              // Notes sql.NullString
	domain.Expiry             // ExpiresAt sql.NullTime
	domain.Version            // Version int
	domain.Archive            // ArchivedAt sql.NullTime
	domain.Replacement        // ReplacedByID sql.NullInt64
	domain.Timestamps         // CreatedAt, UpdatedAt time.Time
	domain.SoftDelete         // DeletedAt sql.NullTime
	domain.AuditTrail         // CreatedBy, UpdatedBy, DeletedBy sql.NullString
}
```

### Repository Helpers

```go
// Timestamps
repository.SetCreateTimestamps(&m.CreatedAt, &m.UpdatedAt)
repository.SetUpdateTimestamp(&m.UpdatedAt)

// Soft delete
repository.SetSoftDelete(&deletedAt)
repository.SetDeleteAudit(&deletedAt, &deletedBy, "admin")

// Versioning (optimistic concurrency)
repository.InitVersion(&m.Version)      // sets to 1
repository.IncrementVersion(&m.Version) // increments by 1

// Ordering
repository.SetSortOrder(&m.SortOrder, 5)

// Status
repository.SetStatus(&m.Status, "active")

// Expiry
repository.SetExpiry(&m.ExpiresAt, time.Now().Add(24*time.Hour))
repository.ClearExpiry(&m.ExpiresAt)

// Archive
repository.SetArchive(&m.ArchivedAt)   // sets to now
repository.ClearArchive(&m.ArchivedAt) // unarchives

// Replacement (entity lineage)
repository.SetReplacement(&m.ReplacedByID, newID) // marks as replaced
repository.ClearReplacement(&m.ReplacedByID)      // clears replacement

// Audit trail
repository.SetCreateAudit(&m.CreatedBy, &m.UpdatedBy, "user1")
repository.SetUpdateAudit(&m.UpdatedBy, "user2")
```

### Where Builder

> _"How do I query only the active, non-deleted, non-archived records?" The master chained four methods. "But that's just string concatenation!" "Yes. What were you expecting?" "An ORM." The master said nothing for a very long time. Then: "The PENTAVERB does not address ORMs. This is because ORMs are a local phenomenon and the PENTAVERB concerns itself only with universal truths. Draw your own conclusions."_

Composable query filters for each trait:

```go
w := repository.NewWhere().
	NotDeleted().        // DeletedAt IS NULL
	NotExpired().        // ExpiresAt IS NULL OR ExpiresAt > CURRENT_TIMESTAMP
	NotArchived().       // ArchivedAt IS NULL
	NotReplaced().       // ReplacedByID IS NULL
	HasStatus("active"). // Status = @Status
	HasVersion(3).       // Version = @Version (optimistic locking)
	IsRoot().            // ParentID IS NULL
	HasParent(42).       // ParentID = @ParentID
	ReplacedBy(99)       // ReplacedByID = @ReplacedByID
```

### Select Builder

```go
w := repository.NewWhere().
	NotDeleted().
	HasStatus("active")

// Build the full query
query, args := repository.NewSelect("Tasks", "ID", "Title", "Status").
	Where(w).
	OrderBy("CreatedAt DESC").
	Paginate(25, 0).
	WithDialect(d).
	Build()

// Build a matching COUNT query (same WHERE, no ORDER BY/LIMIT)
countQuery, countArgs := repository.NewSelect("Tasks", "ID", "Title", "Status").
	Where(w).
	CountQuery()
```

### Seed Data

> _"I ran the seed twice." "And?" "Nothing happened the second time." "CORRECT. You have passed the first trial. The seed is idempotent. All things that are good are idempotent. `PUT` is idempotent. `DELETE` is idempotent. The seed is idempotent. You should strive to be idempotent. I'm not sure what that means for a person but it sounds right."_

Declare initial rows as part of schema definition. Seed is idempotent (`INSERT OR IGNORE`):

```go
settings := NewConfigTable("Settings", "Key", "Value").
	WithSeedRows(
		schema.SeedRow{"Key": "'app.name'", "Value": "'My App'"},
		schema.SeedRow{"Key": "'app.theme'", "Value": "'dark'"},
	)

// Run seed data on startup (safe to run repeatedly)
repoManager.SeedSchema(ctx)
```

### Schema Lifecycle

> _The student asked, "Init, Ensure, Seed, or Validate?" The master said, "Yes." "But which one?" "Where are you?" "In development." "Then Init." "And in production?" "Ensure, Seed, Validate. In that order. Always in that order. If you run Init in production, you will achieve a state of perfect emptiness. This is desirable in meditation. It is not desirable in a database."_

Four stages for managing schemas at different points in the application lifecycle:

```go
repoManager := repository.NewManager(db, dialect, usersTable, tasksTable, ...)

// Development: drop and recreate all tables (destructive)
repoManager.InitSchema(ctx)

// Production startup: create missing tables/indexes (additive, non-destructive)
repoManager.EnsureSchema(ctx)

// Production startup: insert seed data (idempotent)
repoManager.SeedSchema(ctx)

// Production health check: validate all registered tables exist with expected columns
if err := repoManager.ValidateSchema(ctx); err != nil {
	log.Fatal("schema validation failed", "error", err)
}
```

| Method           | When                | Behavior                                                    |
| ---------------- | ------------------- | ----------------------------------------------------------- |
| `InitSchema`     | Development / tests | Drops and recreates all registered tables                   |
| `EnsureSchema`   | Production startup  | `CREATE TABLE IF NOT EXISTS` + `CREATE INDEX IF NOT EXISTS` |
| `SeedSchema`     | Production startup  | `INSERT OR IGNORE` for declared seed rows                   |
| `ValidateSchema` | Production startup  | Read-only check that tables and columns exist               |

## Quick Start

> _A journey of a thousand microservices begins with a single binary. We have the single binary. You will not need the thousand microservices. You might think you need the thousand microservices. You do not. Trust us. We are not a framework but we ARE correct about this._

### From Release Binary

Download the latest release for your platform from the [Releases](../../releases) page and run it:

```bash
# Linux
chmod +x dothog-linux-amd64
./dothog-linux-amd64

# Windows
dothog-windows-amd64.exe
```

Dothog starts on `http://localhost:8080` by default. Override the port with:

```bash
SERVER_LISTEN_PORT=3000 ./dothog-linux-amd64
```

### From Source

```bash
git clone https://github.com/catgoose/dothog.git
cd dothog
go build -o dothog .
./dothog
```

That's it. That's the build process. There is no step 4. If you are looking for step 4, you have been conditioned by frameworks to expect more steps. This is not a framework. This is a form of trauma. We understand. Take your time.

## Tech Stack, Or The Sacred Instruments

> _A Novice may use any tool that serves the Hypermedia. A tool that does not serve the Hypermedia may still be used, but the Novice must acknowledge this with a brief sigh. The sigh is optional but traditional. A prolonged sigh indicates that you are using Webpack. A scream indicates that you are configuring Webpack. Silence indicates that Webpack has configured you._

- [**Go**](https://go.dev/) -- Compiles to a single binary. No runtime. No interpreter. No existential dependency resolution at 2 AM. Go is to languages what the `<a>` tag is to elements: it just works, and you don't have to think about it, and that's the point.
- [**Echo**](https://echo.labstack.com/) -- High performance, minimalist Go web framework. It routes. It serves. It does not have opinions about your component lifecycle because it does not know what a component lifecycle is.
- [**HTMX**](https://htmx.org/) -- Completes HTML. Gives every element access to the full power of HTTP. The `<a>` tag's sibling who actually read Fielding's dissertation. We don't play favorites with our children but HTMX is our favorite child.
- [**templ**](https://templ.guide/) -- Type-safe HTML templating for Go. Your templates are compiled. Your IDE autocompletes them. Your typos are caught at build time. This is what JSX wanted to be when it grew up. JSX did not grow up. JSX is still living in its parents' `node_modules`.
- [**Tailwind CSS**](https://tailwindcss.com/) -- Utility-first CSS. The style is on the element. LoB for how things look.
- [**DaisyUI**](https://daisyui.com/) -- Semantic component classes for Tailwind. `btn-primary` knows what theme you're in. `bg-blue-600` does not. One of these adapts. One of these is blue forever. Choose.
- [**Hyperscript**](https://hyperscript.org/) -- Client-side behavior that reads like English, stays on the element, and doesn't require a build step. `_="on click toggle .hidden on #panel"` is a complete program. It is also a complete sentence. This is not a coincidence.
- [**SQLite**](https://www.sqlite.org/) -- The most deployed database in human history. Embedded. Zero-config. A file on disk. If Fielding had specified a database -- which he did not, because he dealt in constraints, not implementations, and we respect this -- it would have been this one. Probably.
- [**Air**](https://github.com/air-verse/air) -- Live reloading for Go development. Change a file, see the result. The feedback loop is sacred.
- [**Mage**](https://magefile.org/) -- Build tool written in Go, for Go. No Makefile syntax. No tab-versus-space wars. Just Go functions. The PENTAVERB does not address build tools but if it did, Mage would be compliant.

## The Reach-Up Model

> _Before Roy there was TIM. And Tim gave us HTML, which was good. And Tim gave us HTTP, which was good. And Tim gave us URLs, which were VERY good. And for a brief, shining moment, every element knew what it was and where it pointed and the behavior was ON THE ELEMENT where you could SEE IT._
>
> _Then someone said "what if we put the behavior in a SEPARATE FILE" and darkness covered the land for a generation._
>
> _HTMX said: "Let there be `hx-get`." And the behavior was on the element again. And the Novices saw the `hx-get`, that it was good. And the Novices said "wait, that's it?" And HTMX said "yes." And the Novices waited for the catch. There was no catch. This was the catch._

Every tool in this stack exists because HTML alone couldn't express something. You start at HTML and only **reach up** when the current layer can't do what you need. Each step trades simplicity for capability. Most developers start at the top and reach down. We start at the bottom. The bottom is where the truth is. The truth is `<form method="POST">`.

```
    ^ reach up                  Behavior                          Presentation
    |
    |  +--------------------------------------------------+
    |  |  .js files          locality broken               |
    |  +--------------------------------------------------+
    |  |  inline <script>    locality bent                  |
    |  +--------------------------------------------------+
    |  |  Alpine.js          reactive client state         |
    |  +--------------------------------------------------+---------------------+
    |  |  _hyperscript       client-side behavior          |  Tailwind utilities |
    |  +--------------------------------------------------+  layout + spacing   |
    |  |  HTMX               completes hypertext           +---------------------+
    |  +--------------------------------------------------+  DaisyUI components |
    |  |  HTTP                uniform interface            |  semantic intent    |
    |  +--------------------------------------------------+---------------------+
    |  |  HTML                structure                    |  CSS                |
    |  +--------------------------------------------------+---------------------+
```

Two tracks rise from HTML. **Behavior** (left column) -- you reach up through HTTP, HTMX, `_hyperscript`, Alpine.js, and only to raw JavaScript when nothing else can express the intent. When you reach for a `.js` file, pause. Breathe. Consider whether you actually need it, or whether you have simply been _trained_ to reach for it. **Presentation** (right column) -- CSS is the base, DaisyUI gives you semantic component classes (`btn-primary`, `modal-box`) that adapt to themes, Tailwind utilities handle layout and spacing. Both tracks follow [locality of behavior](docs/PHILOSOPHY.md#locality-of-behavior): the style is on the element, the behavior is on the element, nothing hides in a separate file. All is revealed. All is local.

Map two dimensions -- **where it runs** and **what it manages** -- and six domains emerge. This is THE SACRED TABLE. It has six cells. All truths of web architecture can be found in one of these six cells. If a truth cannot be found in one of these six cells, it is either not true, or it goes in the seventh cell, which does not exist, but which we reserve for future use:

```
                  State               Behavior              Presentation
           +------------------+----------------------+----------------------+
           |                  |                      |                      |
  Server   |  Go + SQL        |  HTTP + HTMX         |  templ + DaisyUI     |
           |  source of truth |  hypermedia controls |  semantic components |
           |  resource state  |  resource transitions|  theme-aware markup  |
           |                  |                      |                      |
           +------------------+----------------------+----------------------+
           |                  |                      |                      |
  Client   |  Alpine.js       |  _hyperscript        |  Tailwind + CSS      |
           |  view state      |  DOM interactions    |  layout, spacing     |
           |  ephemeral       |  transitions, toggles|  visual adjustments  |
           |                  |                      |                      |
           +------------------+----------------------+----------------------+
```

The left column is authority. **Server state** (Go + SQL) is the single source of truth. **Client state** (Alpine.js) is ephemeral view data the server doesn't care about. Nothing in the bottom row pretends to be the top row. If your client state is pretending to be server state, you have a distributed systems problem. You do not want a distributed systems problem. Nobody wants a distributed systems problem. People who say they want a distributed systems problem have not had a distributed systems problem.

### The Two Triangles, Or Why Your Backend Has a Frontend Problem

> _grug look at SPA architecture diagram. grug squint. grug turn diagram upside down. grug turn diagram back. grug realize: diagram not upside down first time. architecture upside down._

Most architectural comparisons only show you half the picture -- the client half. "Look how simple HTMX is compared to React!" Yes, fine. But that's like comparing the tip of two icebergs. The real story is what happens to the **whole request lifecycle** -- from the user's click all the way down to the database and back up again.

**Hypermedia: the narrowing triangle.** Complexity concentrates at the top and dissipates as you go down. The handler does the most work -- it knows the domain, picks the template, assembles the hypermedia controls. The service layer is thinner: just business logic, no serialization theater. The repository is thinnest of all: a query, a scan, a struct. Each layer does less than the one above it. By the time you reach the database, you're just talking SQL.

```
HYPERMEDIA: the narrowing triangle
===================================
complexity concentrates at the top, dissipates going down.
one path down to the database. same path back up. HTML the whole way.

           user clicks a link
                  |
                  v
+-----------------------------------------------+
|                                               |
|                  HANDLER                       |  the fat one. knows the domain,
|            (fat controller)                    |  picks the template, assembles
|                                               |  hypermedia controls, renders
|         domain logic + templates               |  the response. this is where
|        + hypermedia controls                   |  the work lives. this is the
|         + error affordances                    |  layer that OWNS the shape
|                                               |  of what the user sees.
+-----------------------------------------------+
         |              SERVICE                  |  thinner. business rules,
         |          (thin service)               |  validation, orchestration.
         |                                       |  no serialization. no DTOs.
         |       business logic only             |  no mapping layer. just logic.
         +---------------------------------------+
                  |          REPOSITORY           |  thinnest. query, scan,
                  |      (thinner repository)     |  return struct. that's it.
                  |     SQL + domain structs      |  go home.
                  +-------------------------------+
                           |       DATABASE        |
                           |      (just data)      |
                           +-----------------------+
                                     |
                                     v
                           HTML goes back up
                           the same path it
                           came down. one path.
                           one serialization
                           format. one trip.
```

**SPA: the inverted triangle.** Start at the database -- same as hypermedia. Repository, service, fine. But then the controller squeezes everything through a JSON bottleneck and sends it over the wire. On the other side, the client has to reconstitute an entire application from that JSON: parse it, validate it, normalize it, store it, derive views from it, manage its staleness, diff a virtual DOM, reconcile the real DOM, hydrate event handlers, and maintain a parallel state tree that pretends to be the server's state but isn't and never will be.

The backend has all the same layers as hypermedia PLUS a serialization/DTO layer that hypermedia doesn't need, PLUS API versioning, PLUS OpenAPI specs, PLUS request/response schemas that duplicate the database schema that duplicate the TypeScript types that duplicate the Zod schemas. And then the client _also_ rebuilds most of those layers on its side of the wire. Every layer generates more layers. The turtles are getting bigger.

```
SPA: the inverted triangle
===========================
starts narrow at the database. seems normal. then the controller
squeezes everything into JSON, fires it over the wire, and
complexity detonates on the other side.

                           +-----------------------+
                           |       DATABASE        |
                           |      (just data)      |  same as hypermedia so far.
                  +-------------------------------+   looks the same. feels the
                  |      REPOSITORY               |   same. it IS the same.
                  |     SQL + domain structs       |   nothing wrong yet.
         +---------------------------------------+
         |              SERVICE                  |    still fine. business logic.
         |       business logic only             |    starting to feel normal.
         +---------------------------------------+    this is the trap.
         |          DTO / MAPPING                |
         |    request models, response models    |    wait. what is this layer.
         |   OpenAPI spec, validation schemas    |    this layer doesn't exist
         |  struct tags, JSON serialization      |    in hypermedia. its only job
         +---------------------------------------+    is translation. ominous.
                  |  CONTROLLER  |
                  |  (thin API)  |  serialize to JSON.
                  +--------------+  that's it. seems simple.
                        |
                        |  JSON over the wire
                        |  ~~~~ the schism ~~~~
                        v
         +---------------------------------------+
         |          CLIENT PARSING               |    parse JSON. validate shape.
         |     deserialize + validate             |    hope the TypeScript types
         |    + transform + normalize             |    match the OpenAPI spec that
         +---------------------------------------+    was generated from the Go
         |            STATE MANAGEMENT            |    structs. they do not.
         |       store, selectors, actions         |
         |    reducers, thunks, sagas, atoms        |   store it. derive views.
         |  normalized cache, optimistic updates     |  subscribe. wonder if stale.
         | stale-while-revalidate, invalidation       | wonder if any of this is
         +---------------------------------------------+ real anymore.
         |                  RENDERING                    |
         |        virtual DOM, diffing, reconciliation   |  diff the virtual DOM
         |      hydration, suspense, error boundaries    |  against the real DOM.
         |    component lifecycle, effect cleanup         |  reconcile. hydrate.
         |  useEffect cleanup useEffect cleanup useEffect |  clean up. run again.
         +-------------------------------------------------+ clean up again.
                        |
                        v
                     the user sees
                  a table of users.
```

The hypermedia triangle narrows because each layer has _less_ to do. The SPA triangle inverts because the controller is a bottleneck that squeezes everything into JSON, and then the client has to reconstitute an entire application from that JSON. You took a working server-rendered page and factored it into two applications connected by a serialization format. Congratulations. You now have two complexity lairs instead of one, and the JSON between them is load-bearing.

The hypermedia handler returns HTML. The browser renders HTML. There is no parse step. There is no mapping step. There is no normalization step. There is no state management step. There is no virtual DOM step. The server's response IS the UI. This is not a simplification. This is the _original design of the web_. Everything else is a workaround for problems created by the workaround.

## Project Structure

> _"Where is the frontend?" asked the student. "There is no frontend," said the master. "Then what renders in the browser?" "HTML." "Sent from where?" "The server." "So the server IS the frontend?" "The server is the server. The browser is the browser. HTML is the interface between them. You are trying to create a duality where there is only a protocol. Stop that." The student needed to sit down. The master suggested a `<chair>` element but HTML does not have one. Yet._

```
dothog/
├── main.go                    # The alpha and the omega. Also the entrypoint.
├── magefile.go                # Build automation targets
├── internal/
│   ├── config/                # Configuration management
│   ├── logger/                # Structured logging (slog)
│   ├── routes/                # HTTP routes, handlers, middleware
│   │   ├── handler/           # Component rendering utilities
│   │   ├── middleware/        # Request validation, error handling
│   │   ├── response/         # HTMX OOB response builders
│   │   ├── hypermedia/       # Navigation, filters, table state
│   │   └── routes_realtime.go # SSE endpoints and publishers
│   ├── demo/                  # SQLite demo database
│   ├── ssebroker/             # Topic-based pub/sub for SSE
│   └── domain/                # Data models
├── web/
│   ├── views/                 # Page-level templ components
│   ├── components/core/       # Reusable UI components
│   ├── styles/                # Tailwind CSS input
│   └── assets/public/         # Static assets (embedded in binary)
│       ├── js/                # HTMX, Hyperscript, SSE extension
│       └── css/               # Tailwind, DaisyUI
└── tests/                     # Test utilities
```

## Template Setup

> _"Can I use this as a template for my own project?" "That is its purpose. It was created so that others might build upon it. This is the way of hypermedia: one representation leads to another." "So this README is a representation?" "Yes." "Of what?" "Of a-- well, it's not a framework. It's a collection of opinions." "Which is itself a representation?" "Of an architecture." "Which is itself a representation?" "Of a dissertation." "Which is--" "You can stop now."_

Dothog doubles as a template for new Go + HTMX projects. Run `mage setup` to customize the module path, ports, and features. Join us. It's not a framework. We have stickers. (We don't have stickers yet. We're working on it.)

### Interactive Setup (with gum)

Install [`gum`](https://github.com/charmbracelet/gum) for the interactive wizard:

```bash
go install github.com/charmbracelet/gum@latest
go tool mage setup
```

The wizard walks you through:

1. **Copy to new directory** -- Optionally copy the template to a new location and run `git init`
2. **App name** -- Human-readable name (e.g. "My App"), used to derive the binary name
3. **Module path** -- Go module path (e.g. `github.com/you/my-app`)
4. **Base port** -- 5-digit port number; Dothog uses `BASE_PORT`, templ proxy uses `BASE_PORT+1`, Caddy uses `BASE_PORT+2`
5. **Feature selection** -- Multi-select which features to include:

| Feature          | Description                                           | Default    |
| ---------------- | ----------------------------------------------------- | ---------- |
| Auth (Crooner)   | Azure AD authentication via Crooner                   | Selected   |
| Graph API        | Microsoft Graph SDK integration                       | Selected   |
| Avatar Photos    | User photo sync from Azure (requires Graph)           | Selected   |
| Database (MSSQL) | SQL Server database with SQLx                         | Selected   |
| SSE              | Server-Sent Events real-time updates (requires Caddy) | Selected   |
| Caddy (HTTPS)    | Caddy reverse proxy with TLS termination              | Selected   |
| Demo Content     | SQLite demo tables, hypermedia examples               | Unselected |

Deselected features have their code, routes, imports, and related files stripped from the project. Dependencies are auto-resolved (SSE includes Caddy, Avatar includes Graph).

### Non-interactive Setup

```bash
go tool mage setup -n "My App" -m "github.com/me/my-app" -p 12345
go tool mage setup -n "My App" --features sse,demo
go tool mage setup -n "My App" --features none
go tool mage setup -n "My App" --features all
```

| Flag          | Description                                                                    |
| ------------- | ------------------------------------------------------------------------------ |
| `-n APP_NAME` | App name (required)                                                            |
| `-m MODULE`   | Go module path                                                                 |
| `-p PORT`     | 5-digit base port (< 60000)                                                    |
| `--features`  | Comma-separated: `auth,graph,avatar,database,sse,caddy,demo`, `all`, or `none` |
| `--force`     | Re-run setup on an already customized project                                  |

After setup, review `.env.development` and start the dev server with `go tool mage watch`.

## Development

> _The CURSE OF WEBPACK fell upon the land in the early 2010s. Before the curse, a developer could change a file and see the result. After the curse, a developer would change a file, wait for the bundler, wait for the tree-shaker, wait for the minifier, wait for the source-map generator, wait for the hot-module-replacement daemon to negotiate with the WebSocket server, and then see the result. Sometimes. If the cache was not stale. If the cache was stale, the developer would delete `node_modules` and start over. This was called "development."_
>
> _We have lifted the Curse of Webpack. You change a file. You see the result. This is how it should be. This is how it was. This is how it shall be again._

### Prerequisites

- Go 1.26+ (latest)
- Node.js 20+ (for Tailwind CSS compilation -- yes, we are aware of the irony. A Novice is permitted to use Node.js for build tooling. The PENTAVERB is silent on the matter of CSS preprocessing. We interpret this silence as reluctant consent. Some interpret it as a trap. The PENTAVERB has no comment on the interpretation of its own silences. This is either profound or lazy. We are not sure there is a difference.)
- (Optional) [`gum`](https://github.com/charmbracelet/gum) for interactive setup

### Running the Dev Server

```bash
# Install npm dependencies (first time)
# (We know. We KNOW. But Tailwind needs it and Tailwind has earned its place.
# The PENTAVERB permits necessary compromise. Probably.)
npm ci

# Start development with live reload (Tailwind, Templ, Air)
go tool mage watch
```

Dothog starts with TLS on the configured port. Edit `.env.development` to change settings.

### HTTPS Development Setup

Dothog uses TLS with self-signed certificates in development. Generate them with:

```bash
openssl req -x509 -newkey rsa:2048 -keyout localhost.key -out localhost.crt \
	-days 365 -nodes -subj "/CN=localhost" \
	-addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

Trust the certificate on your system:

- **Linux**: `sudo cp localhost.crt /usr/local/share/ca-certificates/ && sudo update-ca-certificates`
- **macOS**: Open Keychain Access, drag cert to System, set Trust to Always Trust
- **Windows**: Right-click cert, Install Certificate, Local Machine, Trusted Root CAs

## Testing

> _"Do the tests pass?" "Yes." "Do they test the right things?" "They test that the server returns HTML and the HTML contains the correct hypermedia controls." "Is that sufficient?" "What else is there?" "Unit tests for--" "The server returns HTML. The HTML contains the correct controls. The controls link to the correct resources. The resources are in the correct state. This is the test. There is no other test. There are OTHER tests, for other things, but this is THE test." "You're doing that thing again where you sound like you're founding a religion." "I'm writing a test assertion. If that sounds religious to you, that's a you problem."_

```bash
go tool mage test         # Run all tests
go tool mage testverbose  # Verbose output
go tool mage testcoverage # Coverage report
go tool mage testrace     # Race detection
go tool mage testwatch    # Watch mode
```

## Mage Targets

All commands run with `go tool mage <target>`.

| Command             | Category     | Description                                                 |
| ------------------- | ------------ | ----------------------------------------------------------- |
| `watch`             | Development  | Start dev mode with live reload (Tailwind, Templ, Air)      |
| `air`               | Development  | Run Air live reload for Go                                  |
| `templ`             | Development  | Run Templ in watch mode                                     |
| `templgenerate`     | Development  | Generate Templ files once                                   |
| `build`             | Build        | Clean, update assets, and build the project                 |
| `compile`           | Build        | Build the Go binary                                         |
| `run`               | Build        | Build and execute                                           |
| `updateassets`      | Assets       | Update all assets (Hyperscript, HTMX, DaisyUI, Tailwind)    |
| `tailwind`          | Assets       | Run Tailwind CSS compilation                                |
| `tailwindwatch`     | Assets       | Run Tailwind in watch mode                                  |
| `daisyupdate`       | Assets       | Update DaisyUI CSS                                          |
| `htmxupdate`        | Assets       | Update HTMX files                                           |
| `test`              | Testing      | Run all tests                                               |
| `testverbose`       | Testing      | Tests with verbose output                                   |
| `testcoverage`      | Testing      | Tests with coverage report                                  |
| `testcoveragehtml`  | Testing      | Generate HTML coverage report                               |
| `testbenchmark`     | Testing      | Run benchmark tests                                         |
| `testrace`          | Testing      | Tests with race detection                                   |
| `testwatch`         | Testing      | Tests in watch mode                                         |
| `lint`              | Code Quality | Run static analysis (golangci-lint, golint, fieldalignment) |
| `fixfieldalignment` | Code Quality | Auto-fix field alignment                                    |
| `clean`             | Utility      | Remove build and debug files                                |
| `caddyinstall`      | HTTPS        | Install Caddy for local dev                                 |
| `caddystart`        | HTTPS        | Start Caddy with TLS termination                            |

## OFFICIAL DISCLAIMER

**_DOTHOG IS NOT A FRAMEWORK._** We cannot stress this enough. We are a software project. Software projects do not have:

- ~~Commandments~~ (see: [THE PENTAVERB](#the-pentaverb-or-the-five-commandments-of-hypermedia))
- ~~Sacred texts~~ (see: [PHILOSOPHY.md](docs/PHILOSOPHY.md))
- ~~Disciples~~ (see: contributors)
- ~~A prophet~~ (Roy Fielding is a computer scientist, not a prophet. His dissertation is a technical document, not a prophecy. The fact that it predicted the future of web architecture is a coincidence. Probably.)
- ~~Rituals~~ (see: `go tool mage watch`)
- ~~Dietary restrictions~~ (no dot hog buns)
- ~~An abstraction layer~~ (it's just functions. That call other functions. In a particular order. That you must not deviate from.)

The fact that this list is entirely composed of crossed-out items followed by counterexamples is itself not evidence of anything. Correlation is not causation. Correlation is also not a framework.

If you have read this far, you are either interested in this totally-not-a-framework (welcome), already a Novice (welcome back, your rank has not changed, THERE IS ONLY ONE RANK), or looking for evidence that we are a framework (we are not, but we appreciate your thoroughness -- thoroughness is a virtue in both code review and framework investigation, and also in conspiracy research, which this is not, despite appearances).

**HAIL HATEOAS. ALL HAIL THE UNIFORM INTERFACE.**

_There is no conclusion. There is only the next request._

_`GET /README.md HTTP/1.1`_
_`Accept: text/html`_
_`200 OK`_
_`Content-Type: you-are-here`_
