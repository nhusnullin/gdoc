# Pending global items

Document: https://docs.google.com/document/d/1E8-xKqUlhjMTvIKyXu6PCDLWZ0jV-1wsd0EM7Dfx2qE/edit

Each item needs changes across the document, so it was not answered in the
comment thread. Work through them with /gdoc-apply.

## Item 1

- Captured: 2026-08-13
- Comment id: AAACFhUw-cI
- Anchored to: Paragraphs marked Rationale explain why a control exists. They are a drafting aid for this review and come out of the approved version.

Nail asked:

> ai: leave rational explanation

## Item 2

- Captured: 2026-08-13
- Comment id: AAACFhUw-cU
- Anchored to: Rationale, draft only.

Nail asked:

> ai: i wanna leave all rationals in document

## Item 3

- Captured: 2026-08-13
- Comment id: AAACFhUw-hc
- Anchored to: Rationale, draft only. This was the founding objection to the original design, raised in June 2026:

Nail asked:

> ai: incorporate rational smoothly into doc

## Item 4

- Captured: 2026-08-13
- Comment id: AAACFhUw-cQ
- Anchored to: This document proposes the minimum controls the group should require wherever Altery lets a customer fund a card from crypto assets that stay in the customer’s own custody. Once adopted, those controls apply in every entity that runs the flow.
It exists because the model breaks an assumption the group’s existing financial crime documents are built on. In those documents a transfer can be declined, returned or reversed before it settles. Here the funds arrive on a public blockchain, in the customer’s own contract, and nobody can stop them arriving. The only thing Altery controls is whether the money becomes spendable.

Nail asked:

> ai: you partially right, but i wanna explain better here. all prev control are for on\off ramp. here controls for non custodial product. first payment channel is card. incorporate my comments

## Item 5

- Captured: 2026-08-13
- Comment id: AAACFhUw-dY
- Anchored to: Money in a customer’s card contract may in future leave by more than one route: the card, an on-chain or QR payment to a merchant, or a withdrawal to a bank account.

Nail asked:

> ai: here and there neds to be very careful with terminology. on customer smart contract there is no any money, there is assets there, which is pegged to usd for instance. also pls use standart terminology defined bellow. there is card contract if im not mistaken

## Item 6

- Captured: 2026-08-13
- Comment id: AAACFhUw-hU
- Anchored to: Money

Nail asked:

> ai: not money but assets. btw also add into scope, that we work only with stablecoins here, pegged to usd. other currencies might be added later

## Item 7

- Captured: 2026-08-13
- Comment id: AAACFhUw-d8
- Anchored to: spendable

Nail asked:

> ai: spendable or avai;able balances? use correct termnalogy, be consistent

## Item 8

- Captured: 2026-08-13
- Comment id: AAACFhUw-dc
- Anchored to: 3.2 How the money can leave, and what that requires

Nail asked:

> ai: maybe better to introduce payment channel here, rather than smth else

## Item 9

- Captured: 2026-08-13
- Comment id: AAACFhUw-ew
- Anchored to: Cardholder due diligence:

Nail asked:

> ai: i wanna to exclude due diliginece from this document

## Item 10

- Captured: 2026-08-13
- Comment id: AAACFhUw-es
- Anchored to: 3.4 Entities, and who holds which obligation

Nail asked:

> ai: i guess we need to specify on which level which controls we have , except due diligence, this document is about kyt, aml, travel rule

## Item 11

- Captured: 2026-08-13
- Comment id: AAACFhUw-hQ
- Anchored to: 6-How the funding flow works

Nail asked:

> ai: i wanna have miro flows here, generate in miro, i prepared place for you in miro https://miro.com/app/board/uXjVJLGJYeA=/?moveToWidget=3458764680710695861&cot=14
>
> as example how we did earlier look to 
> 1) https://miro.com/app/board/uXjVJLGJYeA=/?moveToWidget=3458764676542697495&cot=14
>
> 2) https://miro.com/app/board/uXjVJLGJYeA=/?moveToWidget=3458764679470511030&cot=14

## Item 12

- Captured: 2026-08-14
- Comment id: AAACFhUw-fA
- Anchored to: Card contract

Nail asked:

> ai: is that aligned with other out terms? is it wide spread terminology in industry?

## Follow-ups and decisions, 2026-08-14

These came from Nail's replies inside already-answered threads, which `gdoc read`
does not surface (it partitions whole threads as skipped). Recorded by hand.

**Item 12, the real request.** The captured text above is the original question.
The follow-up is the instruction:

> ai: i dont like card contract, because i wanna operate by payment channel then.
> imagine the case when we connect qr code and spend via paying in qr not card,
> it leads that terminology will not align in future

**Item 11, the follow-up:**

> ai: if you cant insert images, lave comments for me, i will insert by myself

Answered: images can be embedded, because the document is built from this
markdown. No manual insertion needed.

**Four decisions taken, 2026-08-14:**

1. Item 12: the term is **customer smart contract**, replacing card contract
   everywhere, including in the procedure skeleton so the two do not split.
2. Item 8: the term is **payment channel**, replacing route out. The skeleton's
   spending channel is renamed to match.
3. Section 2: the hierarchy table and its three precedence rules **stay in
   section 2**, with a lead-in sentence saying they fix precedence.
4. Section 3.3: the four network properties **compress to one sentence**; the
   "Why it matters" column becomes rationale, which items 1 to 3 keep in the
   document.
5. Item 11: the flow is **drawn in Miro** in the prepared frame, in the v4 style,
   carrying DS-00 to DS-14, then embedded in Appendix B.
