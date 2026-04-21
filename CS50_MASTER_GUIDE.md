# CS50 Master Guide

This guide is a full-course map for learning the major CS50 concept set, in sequence.
Use it with the interactive trainer at `./cs50`.

## How to Use This Package

1. Run `./cs50` and complete all 28 checkpoints.
2. For each module below, complete the drill set.
3. Build the three projects at the end.
4. Write a short retrospective after each project: what worked, what broke, what you changed.

## Module 0: Computational Thinking

Concepts:
- binary, decimal, hexadecimal
- bits, bytes, overflow
- text and media encoding
- abstraction and decomposition

Drills:
- Convert 5 random decimals to binary and back.
- Explain why 8 bits can represent 256 values.
- Write a paragraph describing abstraction in everyday software.

## Module 1: C Basics

Concepts:
- source files, compilation, linking
- variables, types, operators
- conditionals and loops
- command-line arguments

Drills:
- Write a C-style pseudocode program that validates input and branches.
- Describe compile-time vs runtime errors.
- Explain why integer division can surprise beginners.

## Module 2: Arrays and Strings

Concepts:
- array indexing and bounds
- string representation and null terminator
- iteration patterns

Drills:
- Manually trace a loop over a string and list each index/value.
- Explain an off-by-one bug and how to prevent it.
- Compare mutable arrays vs immutable string behavior (language-dependent).

## Module 3: Algorithms

Concepts:
- linear vs binary search
- bubble, selection, insertion, merge sort
- Big-O, Omega, Theta intuition

Drills:
- Classify 10 operations by rough growth rate.
- Explain when binary search is invalid.
- Compare merge sort and insertion sort tradeoffs.

## Module 4: Memory

Concepts:
- stack and heap
- pointers, addresses, dereferencing
- `malloc`, `free`, lifetime
- memory leaks and dangling pointers

Drills:
- Draw stack/heap state for a function call sequence.
- Explain two causes of segmentation faults.
- Describe leak detection workflow with a memory checker.

## Module 5: Data Structures

Concepts:
- linked lists
- hash tables
- trees and tries
- queues, stacks, priority queues

Drills:
- Pick a structure for each of 5 scenarios and justify choice.
- Explain collision handling in hash tables.
- Compare tree traversal strategies.

## Module 6: Python

Concepts:
- syntax differences from C
- functions, modules, exceptions
- list/dict/set comprehensions
- file I/O and CSV processing

Drills:
- Re-implement a prior C exercise in Python.
- Parse a CSV and compute summary stats.
- Write tests for one pure function.

## Module 7: SQL and Data Modeling

Concepts:
- tables, rows, keys
- SELECT, WHERE, ORDER BY, GROUP BY
- joins, indexes, normalization
- transactions and constraints

Drills:
- Model a small app schema (users, posts, comments).
- Write one query each for filtering, aggregation, and joining.
- Explain the cost/benefit of adding an index.

## Module 8: Web Foundations

Concepts:
- HTTP requests/responses, status codes
- HTML semantics
- CSS layout and responsive design
- JavaScript and DOM events

Drills:
- Build a static page with semantic sections.
- Add a responsive layout for mobile and desktop.
- Add one interactive JS behavior via event listener.

## Module 9: Flask and Backend

Concepts:
- routing and templates
- forms and validation
- sessions and authentication
- API calls and JSON handling

Drills:
- Implement login/logout flow with session checks.
- Add server-side input validation.
- Build one endpoint returning JSON.

## Module 10: Security and Ethics

Concepts:
- hashing and salting passwords
- SQL injection prevention
- XSS and CSRF mitigation
- privacy, fairness, and responsible design

Drills:
- Explain parameterized queries with a bad vs good example.
- Identify one XSS risk in a sample form workflow.
- Write a privacy checklist for a student app.

## Module 11: Scaling and Systems Thinking

Concepts:
- caching
- load balancing
- horizontal vs vertical scaling
- observability and reliability basics

Drills:
- Sketch a scaled architecture for a class web app.
- Explain cache invalidation tradeoffs.
- Define SLO-style goals for latency and uptime.

## Three Capstone Projects

1. CLI Project:
- Build a command-line utility with argument parsing, file I/O, and tests.
- Must include at least one non-trivial data structure.

2. Data Project:
- Build a dataset-backed app using SQL joins and indexed queries.
- Must include a short performance analysis before/after indexing.

3. Full-Stack Project:
- Build a Flask app with auth, persistent storage, and secure form handling.
- Must include one interactive frontend feature and one API endpoint.

## Mastery Checklist

- I can explain each concept without notes.
- I can implement each concept in a small program.
- I can debug incorrect behavior systematically.
- I can evaluate tradeoffs (speed, memory, complexity, security).
- I can design and ship a small end-to-end project.
