<script lang="ts">
  // A small dot-matrix glyph, like the emblem at the top of the screen in the
  // reference device. `kind` picks which pattern lights up.
  export let kind: string = "grid";

  const SIZE = 9;

  // Weather glyphs as explicit 9x9 bitmaps ('#' = lit). Clearer than math at
  // this resolution.
  const BITMAPS: Record<string, string[]> = {
    sun: [
      "....#....",
      ".#.....#.",
      "....#....",
      "...###...",
      "#.#####.#",
      "...###...",
      "....#....",
      ".#.....#.",
      "....#....",
    ],
    cloud: [
      ".........",
      "...###...",
      "..#####..",
      ".#######.",
      ".#######.",
      "..#####..",
      ".........",
      ".........",
      ".........",
    ],
    rain: [
      "...###...",
      "..#####..",
      ".#######.",
      "..#####..",
      ".........",
      ".#..#..#.",
      "#..#..#..",
      ".#..#..#.",
      ".........",
    ],
    snow: [
      "#...#...#",
      ".#..#..#.",
      "..#.#.#..",
      "...###...",
      "#########",
      "...###...",
      "..#.#.#..",
      ".#..#..#.",
      "#...#...#",
    ],
    storm: [
      ".....##..",
      "....##...",
      "...##....",
      "..######.",
      "...##....",
      "..##.....",
      ".##......",
      ".........",
      ".........",
    ],
    wind: [
      ".........",
      "......#..",
      ".######..",
      ".......#.",
      ".#######.",
      ".....#...",
      ".#####...",
      ".........",
      ".........",
    ],
    lock: [
      ".........",
      "...###...",
      "..#...#..",
      "..#...#..",
      ".#######.",
      ".###.###.",
      ".###.###.",
      ".#######.",
      ".........",
    ],
  };

  // Each pattern is a predicate over (row, col) deciding if a dot is lit.
  function lit(kind: string, r: number, c: number): boolean {
    const bm = BITMAPS[kind];
    if (bm) return bm[r][c] === "#";
    const cx = (SIZE - 1) / 2;
    const dx = c - cx;
    const dy = r - cx;
    const dist = Math.sqrt(dx * dx + dy * dy);
    switch (kind) {
      case "wifi": {
        // concentric arcs (top half), plus a base dot
        if (r === SIZE - 1 && c === cx) return true;
        const ring = Math.round(dist);
        return r <= cx && (ring === 2 || ring === 4) ;
      }
      case "cpu": {
        // a chip: filled core with pins
        const core = r >= 2 && r <= 6 && c >= 2 && c <= 6;
        const pins =
          (r === 0 || r === SIZE - 1) && c % 2 === 0 && c >= 2 && c <= 6;
        const pinsLR =
          (c === 0 || c === SIZE - 1) && r % 2 === 0 && r >= 2 && r <= 6;
        return core || pins || pinsLR;
      }
      case "git": {
        // two nodes joined by a line (branch glyph)
        if (c === 2 && r >= 2 && r <= 6) return true; // trunk
        if (r === 2 && c >= 2 && c <= 6) return true; // branch out
        if (c === 6 && r >= 2 && r <= 4) return true;
        return (r === 6 && c === 2) || (r === 4 && c === 6) || (r === 2 && c === 2);
      }
      case "chat": {
        // speech bubble
        const body = r >= 1 && r <= 5 && c >= 1 && c <= 7;
        const tail = (r === 6 && c === 3) || (r === 7 && c === 2);
        const hollow = r >= 2 && r <= 4 && c >= 2 && c <= 6;
        return (body && !hollow) || tail;
      }
      default: {
        // rounded filled square (the default emblem)
        return dist <= 3.6;
      }
    }
  }

  $: rows = Array.from({ length: SIZE }, (_, r) =>
    Array.from({ length: SIZE }, (_, c) => lit(kind, r, c))
  );
</script>

<div class="matrix" aria-hidden="true">
  {#each rows as row}
    <div class="row">
      {#each row as on}
        <span class="dot" class:on></span>
      {/each}
    </div>
  {/each}
</div>

<style>
  .matrix {
    display: inline-flex;
    flex-direction: column;
    gap: 2px;
  }
  .row {
    display: flex;
    gap: 2px;
  }
  .dot {
    width: 3px;
    height: 3px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.06);
  }
  .dot.on {
    background: var(--lcd-text);
    box-shadow: 0 0 3px rgba(236, 233, 223, 0.5);
  }
</style>
