// Template pour rapport de projet de fin de Baccalauréat
// Faculté des sciences - Département d'informatique

#let project-report(
  title: "Titre du projet",
  team-name: "Nom de l'équipe",
  team-number: "EQ00",
  authors: (),
  date: datetime.today().display("[day] [month repr:long] [year]"),
  logo: none,
  accent-color: rgb("#2563eb"),
  body,
) = {
  // Configuration du document
  set document(title: title, author: authors.map(a => a.name))
  set page(
    paper: "a4",
    margin: (top: 2.5cm, bottom: 2.5cm, left: 2.5cm, right: 2.5cm),
    header: context {
      if counter(page).get().first() > 1 [
        #set text(size: 9pt, fill: luma(100))
        #team-name — #title
        #h(1fr)
        #team-number
        #line(length: 100%, stroke: 0.5pt + luma(200))
      ]
    },
    footer: context {
      set text(size: 9pt, fill: luma(100))
      line(length: 100%, stroke: 0.5pt + luma(200))
      v(0.3em)
      [Rapport de projet de fin de Baccalauréat]
      h(1fr)
      [Page #counter(page).display("1 / 1", both: true)]
    },
  )

  // Typographie
  set text(font: "New Computer Modern", size: 11pt, lang: "fr")
  set par(justify: true, leading: 0.65em)
  set heading(numbering: "1.1")

  // Style des titres
  show heading.where(level: 1): it => {
    pagebreak(weak: true)
    v(1em)
    block(
      width: 100%,
      {
        set text(size: 18pt, weight: "bold", fill: accent-color)
        box(
          width: 4pt,
          height: 1.2em,
          fill: accent-color,
          baseline: 0.2em,
        )
        h(0.5em)
        it.body
      },
    )
    v(0.5em)
    line(length: 100%, stroke: 1pt + accent-color.lighten(60%))
    v(1em)
  }

  show heading.where(level: 2): it => {
    v(1em)
    block({
      set text(size: 14pt, weight: "bold", fill: accent-color.darken(20%))
      it.body
    })
    v(0.5em)
  }

  show heading.where(level: 3): it => {
    v(0.8em)
    block({
      set text(size: 12pt, weight: "semibold", fill: luma(30))
      it.body
    })
    v(0.3em)
  }

  // Style des liens
  show link: it => text(fill: accent-color, it)

  // Style du code
  show raw.where(block: false): box.with(
    fill: luma(245),
    inset: (x: 4pt, y: 2pt),
    outset: (y: 2pt),
    radius: 3pt,
  )

  show raw.where(block: true): block.with(
    fill: luma(248),
    inset: 12pt,
    radius: 6pt,
    width: 100%,
    stroke: 0.5pt + luma(220),
  )

  // === PAGE DE TITRE ===
  {
    set page(header: none, footer: none)

    v(1fr)

    align(center)[
      #if logo != none {
        image(logo, width: 3cm)
        v(1em)
      }

      #text(size: 12pt, fill: luma(80), weight: "medium")[
        Faculté des sciences — Département d'informatique
      ]

      #v(0.5em)
      #line(length: 40%, stroke: 1pt + accent-color)
      #v(2em)

      #text(size: 11pt, fill: luma(100), tracking: 0.1em)[
        RAPPORT DE PROJET DE FIN DE BACCALAURÉAT
      ]

      #v(2em)

      #block(
        width: 85%,
        {
          set text(size: 28pt, weight: "bold", fill: accent-color)
          title
        },
      )

      #v(2em)

      #rect(
        fill: accent-color.lighten(90%),
        stroke: accent-color.lighten(60%),
        radius: 8pt,
        inset: 1.2em,
      )[
        #set text(size: 12pt)
        #text(weight: "bold", fill: accent-color)[Équipe #team-number] — #team-name
      ]

      #v(3em)

      // Auteurs
      #if authors.len() > 0 {
        grid(
          columns: calc.min(authors.len(), 3),
          gutter: 2em,
          ..authors.map(author => [
            #set align(center)
            #text(weight: "semibold", size: 11pt)[#author.name]
            #if "matricule" in author [
              \ #text(size: 10pt, fill: luma(100))[#author.matricule]
            ]
          ])
        )
      }
    ]

    v(1fr)

    align(center)[
      #line(length: 30%, stroke: 0.5pt + luma(200))
      #v(1em)
      #text(size: 11pt, fill: luma(80))[#date]
    ]

    v(2em)
  }

  pagebreak()

  // === TABLE DES MATIÈRES ===
  {
    show outline.entry.where(level: 1): it => {
      v(0.8em, weak: true)
      strong(it)
    }

    outline(
      title: [#text(fill: accent-color)[Table des matières]],
      indent: 1.5em,
      depth: 3,
    )
  }

  // === CONTENU ===
  body
}

// Composants utilitaires

#let info-box(title: none, icon: none, color: rgb("#2563eb"), body) = {
  block(
    width: 100%,
    fill: color.lighten(92%),
    stroke: (left: 4pt + color),
    radius: (right: 6pt),
    inset: 1em,
  )[
    #if title != none {
      text(weight: "bold", fill: color)[#if icon != none [#icon ] #title]
      parbreak()
    }
    #body
  ]
}

#let tech-badge(name, color: rgb("#6b7280")) = {
  box(
    fill: color.lighten(85%),
    stroke: 0.5pt + color.lighten(50%),
    radius: 4pt,
    inset: (x: 8pt, y: 4pt),
  )[
    #text(size: 9pt, weight: "medium", fill: color.darken(20%))[#name]
  ]
}

#let metric-card(value, label, color: rgb("#2563eb")) = {
  box(
    width: 100%,
    fill: white,
    stroke: 1pt + luma(230),
    radius: 8pt,
    inset: 1em,
  )[
    #align(center)[
      #text(size: 24pt, weight: "bold", fill: color)[#value]
      #v(0.3em)
      #text(size: 10pt, fill: luma(100))[#label]
    ]
  ]
}

#let feature-item(title, description) = {
  block(
    width: 100%,
    inset: (y: 0.5em),
  )[
    #text(weight: "semibold")[#title] \
    #text(fill: luma(60), size: 10pt)[#description]
  ]
}

// ============================================================
// DÉBUT DU DOCUMENT
// ============================================================

#show: project-report.with(
  title: "Narrabyte",
  team-name: "Narrabyte",
  team-number: "EQ06",
  authors: (
    (name: "Membre 1", matricule: "12345678"),
    (name: "Membre 2", matricule: "23456789"),
    (name: "Membre 3", matricule: "34567890"),
  ),
  date: "Décembre 2025",
  accent-color: rgb("#1e40af"),
)

= Mise en contexte

La documentation logicielle représente l'un des défis les plus persistants du développement moderne. Malgré son importance critique pour l'adoption, la maintenance et l'évolution des projets, elle demeure systématiquement négligée et rapidement obsolète.

#info-box(title: "Problématique", color: rgb("#dc2626"))[
  Les équipes de développement consacrent un temps considérable à maintenir une documentation souvent inadéquate, créant des barrières à l'onboarding des nouveaux développeurs et freinant la contribution aux projets open source.
]

Face à cette problématique universelle, l'émergence des modèles de langage avancés ouvre de nouvelles possibilités d'automatisation intelligente. Notre projet vise à transformer la documentation d'une corvée manuelle en un processus automatisé et intégré au workflow de développement existant.


= But du projet

L'objectif principal de ce projet est de développer un MVP fonctionnel capable d'analyser automatiquement les changements de code et de générer des suggestions de documentation contextuelle.

== Objectifs mesurables

#grid(
  columns: (1fr, 1fr),
  gutter: 1em,
  metric-card("80%", "Précision cible des suggestions"), metric-card("<30s", "Temps de traitement par suggestion"),
  metric-card("3", "Fournisseurs LLM supportés"), metric-card("3", "Plateformes supportées"),
)

== Objectifs spécifiques

+ *Objectif technique* : Analyser les `git diff` et générer des suggestions de documentation contextuelle avec une précision d'au moins 80% selon les évaluations utilisateur.

+ *Objectif d'intégration* : Créer une interface utilisateur intuitive permettant la révision et l'approbation des suggestions générées.

+ *Objectif d'apprentissage* : Maîtriser l'intégration d'agents LLM multi-fournisseurs dans une architecture desktop moderne.

+ *Objectif de déploiement* : Livrer une application desktop cross-platform packagée et prête à l'utilisation.


= Stack technique

== Technologies principales

#grid(
  columns: (1fr, 1fr),
  gutter: 2em,
  [
    === Framework & Backend
    #v(0.5em)
    #tech-badge("Wails v2", color: rgb("#e11d48"))
    #tech-badge("Go", color: rgb("#00add8"))
    #tech-badge("go-git", color: rgb("#f05032"))

    #v(1em)
    === Frontend
    #v(0.5em)
    #tech-badge("React", color: rgb("#61dafb"))
    #tech-badge("TypeScript", color: rgb("#3178c6"))
  ],
  [
    === Base de données
    #v(0.5em)
    #tech-badge("SQLite", color: rgb("#003b57"))

    #v(1em)
    === Intelligence artificielle
    #v(0.5em)
    #tech-badge("OpenAI", color: rgb("#10a37f"))
    #tech-badge("Anthropic", color: rgb("#d97706"))
    #tech-badge("Google AI", color: rgb("#4285f4"))
  ],
)

== Justification des choix

#info-box(title: "Pourquoi Wails?", color: rgb("#059669"))[
  Wails permet de créer des applications desktop cross-platform avec un backend Go performant et un frontend web moderne, tout en produisant des binaires natifs légers.
]


= Architecture

== Vue d'ensemble

// Placeholder pour diagramme d'architecture
#align(center)[
  #rect(
    width: 80%,
    height: 8cm,
    fill: luma(250),
    stroke: 1pt + luma(200),
    radius: 8pt,
  )[
    #align(center + horizon)[
      #text(fill: luma(150), size: 12pt)[
        _Insérer le diagramme d'architecture ici_
      ]
    ]
  ]
]

== Composants principaux

=== Module d'analyse Git
Responsable de l'extraction et de l'analyse des différences entre branches.

=== Moteur d'agents LLM
Orchestre les appels aux différents fournisseurs de modèles de langage et implémente un flux agentique pour l'exploration des fichiers.

=== Interface de révision
Permet aux utilisateurs de visualiser, modifier et approuver les suggestions générées.

=== Gestionnaire de configuration
Gère les préférences utilisateur, les clés API et l'historique des opérations.


= Résumé des fonctionnalités

== Fonctionnalités principales

#feature-item(
  "Analyse automatique des git diff",
  "Détection intelligente des changements entre branches sélectionnées",
)

#feature-item(
  "Génération contextuelle de documentation",
  "Suggestions pertinentes basées sur l'analyse du code et de la documentation existante",
)

#feature-item(
  "Interface de révision intuitive",
  "Permet la modification et l'approbation des suggestions avant application",
)

#feature-item(
  "Support multi-fournisseurs LLM",
  "Flexibilité dans le choix du modèle de langage (OpenAI, Anthropic, Google)",
)

#feature-item(
  "Création automatique de branches",
  "Les changements approuvés sont automatiquement commités dans une nouvelle branche",
)

== Modèle BYOK (Bring Your Own Key)

#info-box(title: "Sécurité et confidentialité", color: rgb("#7c3aed"))[
  L'application utilise un modèle BYOK où les utilisateurs fournissent leurs propres clés API. Aucun code n'est transmis vers des serveurs tiers — tout le traitement s'effectue localement.
]


= Défis rencontrés

== Défis techniques

=== Intégration multi-fournisseurs LLM
Description du défi rencontré et de la solution mise en place...

=== Performance de l'analyse
Description du défi rencontré et de la solution mise en place...

== Défis organisationnels

Description des défis liés à la gestion de projet, coordination d'équipe, etc.

#info-box(title: "Leçon apprise", color: rgb("#f59e0b"))[
  Inclure ici une leçon importante tirée des défis rencontrés...
]


= Analyse des résultats

== Métriques de performance

#grid(
  columns: (1fr, 1fr, 1fr),
  gutter: 1em,
  metric-card("XX%", "Précision atteinte", color: rgb("#059669")),
  metric-card("XXs", "Temps moyen", color: rgb("#2563eb")),
  metric-card("XX", "Tests réussis", color: rgb("#7c3aed")),
)

== Évaluation des objectifs

#table(
  columns: (auto, 1fr, auto),
  stroke: 0.5pt + luma(200),
  inset: 10pt,
  fill: (_, row) => if row == 0 { rgb("#1e40af").lighten(90%) } else { white },

  [*Objectif*], [*Description*], [*Statut*],
  [Technique], [MVP fonctionnel avec précision ≥80%], [✓ / ✗],
  [Intégration], [Interface avec temps 30s], [✓ / ✗],
  [Apprentissage], [Maîtrise agents LLM multi-fournisseurs], [✓ / ✗],
  [Déploiement], [Application cross-platform packagée], [✓ / ✗],
)

== Retour utilisateurs

Résumé des retours obtenus lors des tests utilisateurs...


= Conclusion

== Synthèse du projet

Résumé des accomplissements et de la valeur apportée par le projet...

== Travaux futurs

- Intégration directe avec l'API GitHub (webhooks, automatisation PR)
- Support de modèles LLM auto-hébergés
- Fonctionnalités de collaboration temps réel

== Remerciements

Nous tenons à remercier [noms des personnes à remercier] pour leur soutien tout au long de ce projet.

#v(2em)

#align(center)[
  #line(length: 50%, stroke: 1pt + luma(200))
  #v(1em)
  #text(fill: luma(100), style: "italic")[
    Projet réalisé dans le cadre du cours MPS \
    Faculté des sciences — Département d'informatique
  ]
]
