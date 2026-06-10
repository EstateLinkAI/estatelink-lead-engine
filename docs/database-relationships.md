# EstateLink Database Relationship Diagram

## Current + Pivot Schema

```mermaid
erDiagram

    USERS ||--o{ ACTIVITY_LOG : creates

    IMPORT_JOBS ||--o{ RAW_LISTINGS : contains

    LISTINGS ||--o{ LEAD_SCORES : has
    LISTINGS ||--o{ PROPERTY_IMAGES : has
    LISTINGS ||--o{ PROPERTY_STRATEGY_SCORES : has

    USERS {
        bigint id PK
        text email
        text password_hash
        text role
        timestamptz created_at
    }

    LISTINGS {
        bigint id PK
        text source_platform
        text source_url
        text title
        text city
        text postcode_area
        text property_type
        int bedrooms
        numeric price
        numeric monthly_rent
        timestamptz created_at
    }

    LEAD_SCORES {
        bigint id PK
        bigint listing_id FK
        int score
        text grade
        jsonb reasons
        timestamptz created_at
    }

    IMPORT_JOBS {
        bigint id PK
        text status
        int total_rows
        int successful_rows
        int failed_rows
        text error_message
        timestamptz created_at
        timestamptz completed_at
    }

    RAW_LISTINGS {
        bigint id PK
        bigint import_job_id FK
        jsonb raw_data
        text status
        text error_message
        timestamptz created_at
    }

    ACTIVITY_LOG {
        bigint id PK
        bigint user_id FK
        text action
        text entity_type
        bigint entity_id
        jsonb metadata
        text ip_address
        text user_agent
        timestamptz created_at
    }

    PROPERTY_IMAGES {
        bigint id PK
        bigint listing_id FK
        text source_url
        int position
        boolean is_primary
        timestamptz created_at
    }

    PROPERTY_STRATEGY_SCORES {
        bigint id PK
        bigint listing_id FK
        text strategy
        int score
        text grade
        jsonb reasons
        timestamptz created_at
    }
```

---

## Core Relationship Meaning

```txt
users
↓
activity_log
```

Users trigger actions such as login, imports, listing ingestion, and admin changes.

```txt
import_jobs
↓
raw_listings
```

An import job can contain many raw listing records.

```txt
listings
↓
lead_scores
```

A listing can have one or more lead score records.

```txt
listings
↓
property_images
```

A listing can have multiple scraped image URLs.

```txt
listings
↓
property_strategy_scores
```

A listing can now have multiple investment strategy scores.

Example strategies:

```txt
buy_to_let
brrrr
flip
buy_and_hold
hmo
development
```

---

## Pivot Direction

The old model was:

```txt
listing
↓
lead score
```

The new model becomes:

```txt
listing / property
├── images
├── lead score
├── buy-to-let score
├── BRRRR score
├── flip score
├── buy-and-hold score
├── HMO score
└── development score
```

This is the foundation for EstateLink becoming a property opportunity intelligence platform rather than just a listings importer.
