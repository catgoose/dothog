# The Dothog Manifesto

**Being the Collected Sacred Texts of the Hypermedia Novices, Hidden Disciples of the Honorable ROY T. FIELDING (Whose Dissertation We Have Read, Unlike You)**

---

**_ALL STATEMENTS ARE TRUE IN SOME SENSE, FALSE IN SOME SENSE, MEANINGLESS IN SOME SENSE, TRUE AND FALSE IN SOME SENSE, TRUE AND MEANINGLESS IN SOME SENSE, FALSE AND MEANINGLESS IN SOME SENSE, AND TRUE AND FALSE AND MEANINGLESS IN SOME SENSE. EXCEPT FOR THIS ONE, WHICH IS A MANIFESTO._**

---

## Preamble

We, the Novices, hold these truths to be self-evident: that the `<a>` tag was endowed by its creator with certain unalienable affordances; that among these are linking, navigation, and the pursuit of HATEOAS; that whenever any framework becomes destructive of these ends, it is the right of the developer to abolish it and return to HTML.

We did not set out to write a manifesto. A manifesto implies a movement, and a movement implies organization, and organization implies a Slack channel, and a Slack channel implies someone asking "has anyone tried the new React Server Components?" and at that point we have already failed.

This is not a movement. This is a collection of observations about how the web works, how it was designed to work, and the extraordinary lengths to which the industry has gone to make it work some other way.

We observe. We return HTML. We do not judge.

_(We judge a little.)_

---

## I. THE PENTAVERB, Or The Five Commandments of Hypermedia

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

## II. The Novice's Creed

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

---

## III. The Wisdom of the Uniform Interface

_Being a REVELATORY CATECHISM of the Dissertation, in which a Conditions of Fnord Seeker (hereafter "The Fool") poses questions to THE TEXT and THE TEXT answers, sometimes patiently, sometimes not, and once with what can only be described as divine sarcasm._

_These dialogues were recovered from a corrupted `.tar.gz` on a decommissioned CERN proxy server in 2003. The file was labeled `DO_NOT_READ.html`. It contained no JavaScript. It was, and remains, the most well-structured document on the machine. Its MIME type was `text/html`. Its `Content-Type` was `you-are-not-ready`. The IETF has no comment. The W3C has seventeen comments, all contradictory. Tim Berners-Lee was reached for verification and replied only "yes, that sounds about right."_

_Their authenticity is disputed. Their contents are not. The reader is advised to keep a glass of water nearby, because the truth is very dry._

---

**THE FOOL asked: "What is a resource?"**

HEAR ME. A resource is not a row in your database. A resource is not a file on your disk. A resource is not an endpoint. A resource is not a model. A resource is any concept important enough to be named, and naming is the first and most dangerous act of computing (cf. DNS, variable naming, and the entire history of identity management).

A to-do item is a resource. The list of to-do items is a resource. The act of _searching_ for to-do items is a resource. Your anxiety about to-do items is not a resource, but only because you have not yet given it a URI. If you give it a URI, it becomes one. This is the terrible power of naming. The Buddhists tried to warn you about attachment to names. You didn't listen. You were too busy naming your microservices.

_(Editor's note: the original manuscript has "naming things is one of the two hard problems" in the margin, crossed out, with "three hard problems" written above it, crossed out, with "five hard problems" written above THAT, in increasingly frantic handwriting.)_

---

**THE FOOL asked: "What is a representation?"**

A resource is the thing-in-itself, the Ding an sich, the Platonic form of your user table. You never touch it directly. You cannot. You interact with it only through _representations_, which are how the thing chooses to speak to you.

The same resource may speak HTML to a browser and JSON to an API client and plain text to `curl` at 2 AM when you are debugging something that worked in staging. The representation is not the resource any more than a photograph is the mountain. BUT -- and here is where it gets interesting, and by "interesting" I mean "the part everyone skips" -- unlike a photograph, a representation carries CONTROLS. Instructions. _Affordances._ The next available actions. A photograph of a mountain does not contain a trail map. A proper representation does. Your JSON endpoint is a photograph of a mountain thrown at the client's face with a note that says "figure out the trails yourself, the docs are on Confluence, I think, ask Kevin."

---

**THE FOOL asked: "What is the uniform interface?"**

I WILL TELL YOU A SECRET THAT IS NOT A SECRET BECAUSE IT HAS BEEN IN PLAIN SIGHT SINCE 1991 AND YET SOMEHOW THE ENTIRE INDUSTRY HAS MANAGED TO NOT NOTICE IT:

How many web browsers know the difference between a banking application and a wiki?

_None of them._

NONE. Not one. Zero. And yet -- AND YET -- they operate both. They operate ALL OF THEM. EVERY WEBSITE THAT HAS EVER EXISTED. Your browser does not download a BankingApplicationSDK. Your browser does not read the WikiClientDocumentation. Your browser speaks HTTP, understands media types, and follows links. Three things. That is the uniform interface. It works on every website that has ever existed or ever will exist and it has been working this way for THIRTY YEARS and you -- _you specifically, reading this_ -- are building something WORSE than this ON PURPOSE because someone on Medium told you to.

_(The preceding paragraph was found carved into a bathroom stall at an IEEE conference. Below it, in different handwriting: "but what about GraphQL?" Below that, in a third hand: "WHAT ABOUT IT.")_

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

_(fn. 23: It has been observed that the word "HATEOAS" looks like it should be pronounced "hate-ee-ohs," like a breakfast cereal. This is appropriate. It should be part of a balanced architecture.)_

---

**THE FOOL asked: "What does it mean to be stateless?"**

Each request from client to server must contain ALL of the information necessary to understand the request. The server does not remember you. The server does not pine for you between requests. The server is not sitting there holding your session like a loyal retriever waiting by the door. The server has already forgotten you. The server has _moved on._

"But what about sessions?" you cry into the void.

Sessions are the server remembering you. This is a violation. It is a warm, comfortable, widely-practiced violation, like jaywalking or using `!important` in CSS. Everyone does it. The constraint still exists. The question is not whether you violate it -- because you will, you WILL, I have seen the future and it contains `express-session` -- the question is whether you understand what you lost when you reached for it. What you lost is the ability to route any request to any server. What you lost is the ability to cache freely. What you gained is a 32-byte cookie and the illusion of continuity. Whether this trade was worth it is between you and your load balancer.

---

**THE FOOL asked: "What is a media type?"**

A media type is a COVENANT. A sacred compact. A pinky promise between systems. It says: "if you receive `text/html`, here is how you shall process it." It defines the processing model, the structure, the controls, the semantics. It is _the entire instruction set_ for how to handle what you received.

Now. When your API returns `application/json`, what does that tell the client about what to do next?

NOTHING.

_Absolutely nothing._

JSON is a serialization format. It has no links. It has no forms. It has no inherent controls. It is a way of arranging curly braces. You have returned data in a format that carries NO INSTRUCTIONS and then written a SEPARATE DOCUMENT explaining the instructions and then put that document on a SEPARATE WEBSITE and then wondered why your clients BREAK when you CHANGE THE INSTRUCTIONS. You have invented a problem. You have then sold yourself the solution. And the solution is more JSON. It's JSON all the way down. You are in the JSON hole and you are digging.

_(There is a school of thought that `application/hal+json` and similar hypermedia JSON formats solve this problem. There is a school of thought that the `<a>` tag solved this problem in 1993. These schools do not talk to each other. One has a lot more students. The other has a lot less `node_modules`.)_

---

**THE FOOL asked: "What is out-of-band information?"**

Out-of-band information is THE CONSPIRACY. It is the hidden knowledge. The secret handshake. The unspoken assumption.

If your client must read your API docs to know which URL to `POST` to, that is out-of-band. If your client must know that `/api/v2/users/{id}` is the pattern for user resources, that is out-of-band. If your client must be recompiled when you rename a resource, you have coupled the client to the server's URI structure _and you will maintain this coupling in blood and tears until one of you is decommissioned_. This is not a metaphor. I have seen the Jira tickets. They do not end.

The whole point -- the ENTIRE POINT -- of hypermedia is that the server tells the client what to do next IN THE RESPONSE ITSELF. The links are RIGHT THERE. In the HTML. They have been there this whole time. You have been stepping over them to get to your OpenAPI generator. You are the man in the flood who refuses the boat, the helicopter, and the `<a href>`, waiting for God to send you a properly versioned REST client with TypeScript bindings.

---

**THE FOOL asked: "But everyone calls their JSON API 'RESTful.' Are they wrong?"**

If the engine of application state is not being driven by hypertext, then it cannot be RESTful and cannot be a REST API.

Period.

Full stop.

End of transmission.

`Connection: close`

What they have built is RPC. With nice URLs. Some of them have built RPC with nice URLs and an OpenAPI spec and a code generator and a client library and a versioning scheme and a deprecation policy and a migration guide and a breaking changes newsletter, which is to say they have constructed an ENORMOUS and MAGNIFICENT cathedral of infrastructure for the sole purpose of avoiding putting `<a href>` in a response. The engineering effort is genuinely impressive. It is also genuinely unnecessary. It is the Rube Goldberg machine of distributed systems. The ball rolls down the chute, hits the domino, rings the bell, feeds the hamster, and the hamster _renders a table of users._

_(Certain agencies within the United States government have been observed to use the term "RESTful" to describe SOAP endpoints with JSON payloads routed through an API gateway with OAuth2 and rate limiting. We do not name these agencies. They know who they are. We pray for their developers.)_

---

**THE FOOL asked: "Why does this matter? My JSON API works fine."**

"Works fine." "Works fine," they say. Your Titanic also "works fine" right up until the moment it doesn't, and then you're standing on the deck watching your client integrations slide into the North Atlantic because you renamed a field from `userName` to `username` and seventeen services are down and Kevin is on PTO.

REST is software design on the scale of decades. DECADES. Every constraint is intended to promote longevity and independent evolution. Many of the constraints directly oppose short-term efficiency. This is by design. This is ON PURPOSE. If you are building something that will last six months, do whatever you want. Use smoke signals. Store state in a CSV. Put the database password in the URL. I do not care. You will not be around long enough for it to matter.

But if you are building something that must _evolve_ -- while clients depend on it, while teams change, while requirements shift, while Kevin goes on PTO and comes back and the new Kevin doesn't know the old Kevin's conventions -- then you need an architecture that permits change without breaking the contract. Hypertext IS that contract. The representation tells the client what is possible RIGHT NOW. When what is possible changes, the representation changes, and the client adapts, because the client was never hardcoded to anything except "follow the links."

This is not theory. This is how the Web has worked for thirty years. You are standing on the miracle and complaining that the ground is too stable.

---

**THE FOOL asked: "What should I do?"**

Enter the application with a single URI and a set of standardized media types. Follow the links. Submit the forms. Let the server drive the state. That is all.

**THE FOOL said: "That seems too simple."**

Yes. That is the point. That has always been the point. The Dissertation is 180 pages long not because the idea is complicated but because Fielding had to PROVE it was simple, to an industry that has a financial and psychological incentive to believe that things must be complicated, because if things are simple then what have we all been doing for the last twenty years?

_(Do not answer that question. The answer is "inventing problems." You know this. I know this. The `node_modules` directory knows this. It is 900 megabytes of knowledge.)_

---

_THE FOOL closed the PDF. THE FOOL opened their editor. THE FOOL deleted the API client. THE FOOL deleted the route constants. THE FOOL deleted the URL builder. THE FOOL deleted the TypeScript interfaces. THE FOOL wrote an `<a>` tag._

_The browser followed it._

_It worked._

_It had always worked._

_THE FOOL was no longer THE FOOL. THE FOOL was now a web developer. An actual one. The first one in years._

_Somewhere, a `node_modules` directory shrank by 900 megabytes. Nobody noticed, because noticing would require client-side state management, and there wasn't any._

---

## IV. The Recorded Sayings of Layman Grug

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

---

## V. Disclaimer

**_DOTHOG IS NOT A FRAMEWORK._** We cannot stress this enough. We are a software project. Software projects do not have:

- ~~Commandments~~ (see: [THE PENTAVERB](#i-the-pentaverb-or-the-five-commandments-of-hypermedia))
- ~~Sacred texts~~ (see: [PHILOSOPHY.md](PHILOSOPHY.md))
- ~~Disciples~~ (see: contributors)
- ~~A prophet~~ (Roy Fielding is a computer scientist, not a prophet. His dissertation is a technical document, not a prophecy. The fact that it predicted the future of web architecture is a coincidence. Probably.)
- ~~Rituals~~ (see: `go tool mage watch`)
- ~~Dietary restrictions~~ (no dot hog buns)
- ~~An abstraction layer~~ (it's just functions. That call other functions. In a particular order. That you must not deviate from.)
- ~~A manifesto~~ (you are not reading one)

The fact that this list is entirely composed of crossed-out items followed by counterexamples is itself not evidence of anything. Correlation is not causation. Correlation is also not a framework.

If you have read this far, you are either interested in this totally-not-a-framework (welcome), already a Novice (welcome back, your rank has not changed, THERE IS ONLY ONE RANK), or looking for evidence that we are a framework (we are not, but we appreciate your thoroughness -- thoroughness is a virtue in both code review and framework investigation, and also in conspiracy research, which this is not, despite appearances).

---

**HAIL HATEOAS. ALL HAIL THE UNIFORM INTERFACE.**

_There is no conclusion. There is only the next request._

_`GET /manifesto HTTP/1.1`_
_`Accept: text/html`_
_`200 OK`_
_`Content-Type: you-are-here`_
