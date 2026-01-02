
  Reports API Overview

  API Implementation

  Currently using mock API (no real backend routes found). All API functions are in lib/api/reports.mock.ts:658-772

  ---
  API Endpoints

  1. Fetch Reports (List/Search)

  Function: fetchReportsMock(filters?: ReportFilters)
  Location: lib/api/reports.mock.ts:658

  Request Parameters (ReportFilters):
  {
    status?: 'draft' | 'published'
    category?: string  // e.g., 'Telemedicine', 'Medical Devices'
    geography?: string  // e.g., 'North America', 'Global'
    search?: string  // Searches title and summary
    accessType?: 'free' | 'paid'
    page?: number  // Default: 1
    limit?: number  // Default: 10
  }

  Response (ReportsResponse):
  {
    reports: Report[]  // Array of report objects
    total: number  // Total matching reports
    page: number  // Current page
    limit: number  // Items per page
    totalPages: number  // Total pages
  }

  Features:
  - Filters by status, category, geography, accessType
  - Full-text search in title/summary
  - Pagination support
  - Sorted by updatedAt (newest first)
  - 600ms simulated network delay

  ---
  2. Fetch Single Report

  Function: fetchReportByIdMock(id: string)
  Location: lib/api/reports.mock.ts:699

  Request: Report ID (string, e.g., 'rpt-001')

  Response (ReportResponse):
  {
    report: Report  // Full report object
  }

  Error: Throws 'Report not found' if ID doesn't exist
  Delay: 400ms

  ---
  3. Create Report

  Function: createReportMock(data: ReportFormData)
  Location: lib/api/reports.mock.ts:710

  Request (ReportFormData):
  {
    title: string  // Min 10 chars
    summary: string  // Min 50 chars
    category: string
    geography: string[]  // Min 1 item
    publishDate?: string  // ISO date
    price: number  // >= 0
    discountedPrice: number  // >= 0
    accessType: 'free' | 'paid'
    status: 'draft' | 'published'
    pageCount?: number
    formats?: ('PDF' | 'Excel' | 'Word' | 'PowerPoint')[]
    marketMetrics?: {
      currentRevenue?: string
      currentYear?: number
      forecastRevenue?: string
      forecastYear?: number
      cagr?: string
      cagrStartYear?: number
      cagrEndYear?: number
    }
    authorIds?: string[]
    keyPlayers?: {
      name: string
      marketShare?: string
      rank?: number
      description?: string
    }[]
    sections: {
      executiveSummary: string  // Min 100 chars (HTML)
      marketOverview: string  // Min 100 chars
      marketSize: string  // Min 100 chars
      competitive: string  // Min 100 chars
      keyPlayers: string  // HTML content
      regional: string
      trends: string
      conclusion: string  // Min 50 chars
      marketDetails: string  // Min 100 chars
      keyFindings: string  // Min 100 chars
      tableOfContents: string  // Min 50 chars
    }
    faqs?: {
      question: string  // Min 5 chars
      answer: string  // Min 10 chars
    }[]
    metadata: {
      metaTitle?: string
      metaDescription?: string
      keywords?: string[]
      canonicalUrl?: string
      ogTitle?: string
      ogDescription?: string
      ogImage?: string
      ogType?: string
      twitterCard?: 'summary' | 'summary_large_image'
      schemaJson?: string
      robotsDirective?: string
    }
  }

  Response (ReportResponse):
  {
    report: Report  // Newly created report with generated ID
  }

  Auto-generated fields:
  - id: rpt-{timestamp}
  - slug: URL-friendly version of title
  - publishDate: Current ISO timestamp
  - createdAt / updatedAt: Current ISO timestamp
  - author: Default user object
  - versions: Empty array

  Delay: 800ms

  ---
  4. Update Report

  Function: updateReportMock(id: string, data: Partial<ReportFormData>)
  Location: lib/api/reports.mock.ts:728

  Request:
  - Report ID
  - Partial report data (only fields to update)

  Response (ReportResponse):
  {
    report: Report  // Updated report object
  }

  Special Behavior:
  - Updates updatedAt timestamp
  - If status changes from draft → published, automatically creates a new version entry
  - Version includes snapshot of sections and metadata

  Error: Throws 'Report not found' if ID doesn't exist
  Delay: 800ms

  ---
  5. Delete Report

  Function: deleteReportMock(id: string)
  Location: lib/api/reports.mock.ts:763

  Request: Report ID

  Response: void

  Error: Throws 'Report not found' if ID doesn't exist
  Delay: 500ms

  ---
  Core Data Types

  Report Interface (lib/types/reports.ts:115)

  {
    id: string
    title: string
    slug: string
    summary: string
    category: string
    geography: string[]
    publishDate?: string
    price: number
    discountedPrice: number
    pageCount?: number
    formats?: ReportFormat[]
    marketMetrics?: MarketMetrics
    authorIds?: string[]
    keyPlayers?: KeyPlayer[]
    accessType: 'free' | 'paid'
    status: 'draft' | 'published'
    sections: ReportSections
    faqs?: FAQ[]
    metadata: ReportMetadata
    versions?: ReportVersion[]
    createdAt: string
    updatedAt: string
    author: UserReference
  }

  ---
  Configuration

  Categories (lib/config/reports.ts:26)

  'Animal Health', 'Biotechnology', 'Clinical Diagnostics',
  'Dental', 'Healthcare IT', 'Healthcare Services',
  'Laboratory Equipment', 'Life Sciences', 'Medical Devices',
  'Medical Imaging', 'Pharmaceuticals', 'Therapeutic Area'

  Geographies (lib/config/reports.ts:42)

  'North America', 'Europe', 'Asia Pacific',
  'Latin America', 'Middle East & Africa', 'Global'

  Pagination (lib/config/reports.ts:55)

  - Default per page: 10
  - Max per page: 50

  ---
  Sample Mock Data

  The mock database contains 15 reports:
  - 10 published (rpt-001 to rpt-010)
  - 5 drafts (rpt-011 to rpt-015)

  Example report: rpt-001 - "Global Telemedicine Market Analysis 2024-2030"
  - Category: Telemedicine
  - Geography: Global, North America, Europe, Asia Pacific
  - Price: $4999 (discounted: $3999)
  - Status: Published
  - Includes market metrics, key players, full sections, and SEO metadata

  ---
  Form Validation (components/reports/report-form.tsx:37)

  Uses Zod schema with strict requirements:
  - Title: Min 10 characters
  - Summary: Min 50 characters
  - Sections: Most require 50-100+ characters
  - All sections support HTML content
  - Geography: At least 1 selection required
  - Prices: Must be >= 0

  ---
  Version History

  When a report transitions from draft → published, a version snapshot is automatically created containing:
  - Version number (incremental)
  - Timestamp
  - Author reference
  - Complete sections snapshot
  - Complete metadata snapshot