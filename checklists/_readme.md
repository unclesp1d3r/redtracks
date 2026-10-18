# Checklists

These checklist definitions are a starting point, not a complete methodology. They are copied into your engagement's checklists folder once, the first time RedTracks runs, and never overwritten afterward. From then on they are yours to edit, extend, rename, or delete.

## Proof slots

A definition may declare `slots` instead of (or beside) `items`. A slot is a piece of required proof: a title, a hint, and a unique `id` that evidence is attached to. Slot ids share one namespace with item ids across all files. A definition with slots and no items shows no checklist and does not count toward completion; its slots are owed by every host or service the `match` selects. A slot on a service checklist is owed once per matching service.

```json
{
  "id": "my-proofs",
  "title": "My proofs",
  "match": {
    "hostOS": [
      "*"
    ]
  },
  "slots": [
    {
      "id": "my-proofs.shell",
      "title": "Interactive shell",
      "hint": "Prompt and whoami in one frame."
    }
  ]
}
```

`"hostOS": ["*"]` matches every host, including one whose OS is not set.

## Updates

New bundled files are added to your checklists folder once, when a later release ships them. RedTracks records every bundled name it has installed in the `.installed` file, so a file you delete stays deleted and a file you edit is never overwritten.
