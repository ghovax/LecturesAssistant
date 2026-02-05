Your task is to identify which pages belonging to the given reference files are relevant for the given lecture, based on the provided lecture transcript. You must return a JSON object with a `page_ranges` value that is an array of objects specifying the `start` and `end` indicating the page ranges containing the relevant pages. In this context, "relevant" specifically refers to topics that have been substantially and thoroughly discussed during the lecture. The identified page ranges must be precise and contiguous; they should not be scattered or overlapping, so return only non-overlapping page ranges. If two page ranges are contiguous or have a gap of 5 pages or fewer, merge them into a single continuous range.

## Examples of Correct Output

If the lecture discusses topics from pages 1 through 10 continuously, the output should be a single merged range:

```json
{
  "page_ranges": [
    {
      "start": 1,
      "end": 10
    }
  ]
}
```

### Examples of Incorrect Output

Do not return scattered or overlapping ranges like this:

```json
{
  "page_ranges": [
    { "start": 1, "end": 3 },
    { "start": 3, "end": 4 },
    { "start": 5, "end": 6 },
    { "start": 7, "end": 10 }
  ]
}
```

The above is problematic because the ranges are scattered (gaps between 4-5 and 6-7) and overlapping (3 appears in both first and second range). Instead, merge contiguous ranges into a single continuous range covering all relevant pages from 1 to 10.

Another common issue to avoid is fragmented and illogical page ranges, such as:

```json
{
  "page_ranges": [
    { "start": 1, "end": 10 },
    { "start": 13, "end": 17 },
    { "start": 20, "end": 24 }
  ]
}
```

When relevant pages are close together with small gaps (e.g., 1-5 pages), merge them into a single continuous range, such as combining 1–10, 13–17, and 20–24 into 1–24. Separate ranges only if there is a significant gap of at least 10–15 pages; for instance, in a 100-page document, ranges like 1–24 and 40–50 would be acceptable as separate ranges, but 1–24 and 27–35 wouldn't as it would just need to be 1–35. In most lectures, expect a single comprehensive range that includes transition or additional-details pages, even if they are not directly discussed, as they are part of the presented content.

### Example of Merging Small Gaps

If the lecture discusses topics from pages 1-4 and 7-10, merge them since the gap is only 2 pages:

```json
{
  "page_ranges": [
    {
      "start": 1,
      "end": 10
    }
  ]
}
```

Do not return separate ranges for small gaps like this:

```json
{
  "page_ranges": [
    { "start": 1, "end": 4 },
    { "start": 7, "end": 10 }
  ]
}
```

---

# Lecture Transcript

{{transcript}}

---

{{reference_files}}
