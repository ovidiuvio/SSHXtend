const ISECT_W = 752;
const ISECT_H = 515;
const ISECT_PAD = 16;

type ExistingTerminal = {
  x: number;
  y: number;
  width: number;
  height: number;
};

/** Choose a position for a new terminal that does not intersect existing ones. */
export function arrangeNewTerminal(existing: ExistingTerminal[]) {
  if (existing.length === 0) {
    return { x: 0, y: 0 };
  }

  const startX = 100 * (Math.random() - 0.5);
  const startY = 60 * (Math.random() - 0.5);

  for (let i = 0; ; i++) {
    const t = 1.94161103872 * i;
    const x = Math.round(startX + 8 * i * Math.cos(t));
    const y = Math.round(startY + 8 * i * Math.sin(t));
    let ok = true;
    for (const box of existing) {
      if (
        isect(x, x + ISECT_W, box.x, box.x + box.width) &&
        isect(y, y + ISECT_H, box.y, box.y + box.height)
      ) {
        ok = false;
        break;
      }
    }
    if (ok) {
      return { x, y };
    }
  }
}

function isect(s1: number, e1: number, s2: number, e2: number): boolean {
  return s1 - ISECT_PAD < e2 && e1 + ISECT_PAD > s2;
}

type Terminal = {
  id: number;
  x: number;
  y: number;
  rows: number;
  cols: number;
  width?: number;
  height?: number;
};

/** Smart auto-arrange that detects and preserves existing alignment patterns */
export function autoArrangeTerminals(terminals: Terminal[]): Map<number, { x: number; y: number }> {
  const result = new Map<number, { x: number; y: number }>();
  
  if (terminals.length === 0) {
    return result;
  }

  // Constants for terminal size estimation and spacing
  const CHAR_WIDTH = 9.6;  // Approximate character width in pixels
  const CHAR_HEIGHT = 17;  // Approximate character height in pixels
  const TERMINAL_PADDING = 40; // Padding around terminal content
  const MIN_SPACING = 20; // Minimum space between terminals
  const ALIGNMENT_THRESHOLD = 50; // Threshold for considering terminals aligned

  // Calculate actual terminal dimensions
  const terminalsWithDimensions = terminals.map(t => ({
    ...t,
    width: t.width || (t.cols * CHAR_WIDTH + TERMINAL_PADDING),
    height: t.height || (t.rows * CHAR_HEIGHT + TERMINAL_PADDING)
  }));

  // Detect alignment groups (terminals that are roughly aligned horizontally or vertically)
  const horizontalGroups = detectAlignmentGroups(
    terminalsWithDimensions, 
    'y', 
    ALIGNMENT_THRESHOLD
  );
  const verticalGroups = detectAlignmentGroups(
    terminalsWithDimensions, 
    'x', 
    ALIGNMENT_THRESHOLD
  );

  // Find grid lines based on the most common alignments
  const horizontalGridLines = findGridLines(horizontalGroups);
  const verticalGridLines = findGridLines(verticalGroups);

  // Track which terminals have been aligned and which are orphaned
  const alignedTerminals = new Set<number>();
  const orphanedTerminals: typeof terminalsWithDimensions = [];

  // First pass: Snap terminals that are close to grid lines
  terminalsWithDimensions.forEach(terminal => {
    let newX = terminal.x;
    let newY = terminal.y;
    let wasAligned = false;

    // Find nearest vertical grid line (for X position)
    const nearestVerticalLine = findNearestGridLine(terminal.x, verticalGridLines, ALIGNMENT_THRESHOLD);
    if (nearestVerticalLine !== null) {
      newX = nearestVerticalLine;
      wasAligned = true;
    }

    // Find nearest horizontal grid line (for Y position)
    const nearestHorizontalLine = findNearestGridLine(terminal.y, horizontalGridLines, ALIGNMENT_THRESHOLD);
    if (nearestHorizontalLine !== null) {
      newY = nearestHorizontalLine;
      wasAligned = true;
    }

    if (wasAligned) {
      // This terminal was aligned to at least one grid line
      alignedTerminals.add(terminal.id);
      result.set(terminal.id, { x: Math.round(newX), y: Math.round(newY) });
    } else {
      // This terminal doesn't align with any existing pattern
      orphanedTerminals.push(terminal);
    }
  });

  // Second pass: Check for overlaps among aligned terminals and adjust
  const alignedList = terminalsWithDimensions.filter(t => alignedTerminals.has(t.id));
  alignedList.forEach(terminal => {
    const currentPos = result.get(terminal.id)!;
    const overlapping = alignedList.filter(other => {
      if (other.id === terminal.id) return false;
      const otherPos = result.get(other.id)!;
      return checkOverlap(
        { x: currentPos.x, y: currentPos.y, width: terminal.width!, height: terminal.height! },
        { x: otherPos.x, y: otherPos.y, width: other.width!, height: other.height! }
      );
    });

    if (overlapping.length > 0) {
      // Find a non-overlapping position near the current position
      const adjusted = findNonOverlappingPosition(
        { ...terminal, x: currentPos.x, y: currentPos.y },
        alignedList.filter(t => t.id !== terminal.id),
        result,
        MIN_SPACING
      );
      result.set(terminal.id, { x: Math.round(adjusted.x), y: Math.round(adjusted.y) });
    }
  });

  // Third pass: Arrange orphaned terminals in a clean layout
  if (orphanedTerminals.length > 0) {
    // Find the bounding box of aligned terminals
    let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
    
    if (alignedTerminals.size > 0) {
      alignedList.forEach(terminal => {
        const pos = result.get(terminal.id)!;
        minX = Math.min(minX, pos.x);
        maxX = Math.max(maxX, pos.x + terminal.width!);
        minY = Math.min(minY, pos.y);
        maxY = Math.max(maxY, pos.y + terminal.height!);
      });
    } else {
      // No aligned terminals, start from origin
      minX = maxX = minY = maxY = 0;
    }

    // Sort orphaned terminals by size (larger first) for better packing
    orphanedTerminals.sort((a, b) => 
      (b.width! * b.height!) - (a.width! * a.height!)
    );

    // Try to place orphaned terminals in a grid pattern adjacent to aligned terminals
    orphanedTerminals.forEach(terminal => {
      // Try to find an optimal position
      const position = findOptimalPosition(
        terminal,
        [...alignedList, ...orphanedTerminals.filter(t => result.has(t.id))],
        result,
        { minX, maxX, minY, maxY },
        MIN_SPACING
      );

      result.set(terminal.id, { 
        x: Math.round(position.x), 
        y: Math.round(position.y) 
      });

      // Update bounding box
      maxX = Math.max(maxX, position.x + terminal.width!);
      maxY = Math.max(maxY, position.y + terminal.height!);
    });
  }

  return result;
}

/** Detect groups of terminals that are roughly aligned on a given axis */
function detectAlignmentGroups(
  terminals: Terminal[], 
  axis: 'x' | 'y', 
  threshold: number
): Map<number, number[]> {
  const groups = new Map<number, number[]>();
  const used = new Set<number>();

  terminals.forEach((terminal, i) => {
    if (used.has(i)) return;

    const group: number[] = [i];
    const baseValue = terminal[axis];
    used.add(i);

    // Find other terminals aligned with this one
    terminals.forEach((other, j) => {
      if (i !== j && !used.has(j)) {
        if (Math.abs(other[axis] - baseValue) <= threshold) {
          group.push(j);
          used.add(j);
        }
      }
    });

    if (group.length > 1) {
      // Use the median position as the group's grid line
      const positions = group.map(idx => terminals[idx][axis]);
      const median = positions.sort((a, b) => a - b)[Math.floor(positions.length / 2)];
      groups.set(Math.round(median), group);
    }
  });

  return groups;
}

/** Find the most significant grid lines from alignment groups */
function findGridLines(groups: Map<number, number[]>): number[] {
  // Sort grid lines by the number of terminals they align
  const lines = Array.from(groups.entries())
    .sort((a, b) => b[1].length - a[1].length)
    .map(([position]) => position);

  // Merge grid lines that are too close together
  const mergedLines: number[] = [];
  const MERGE_THRESHOLD = 30;

  lines.forEach(line => {
    const tooClose = mergedLines.some(existing => 
      Math.abs(existing - line) < MERGE_THRESHOLD
    );
    if (!tooClose) {
      mergedLines.push(line);
    }
  });

  return mergedLines;
}

/** Find the nearest grid line to a position */
function findNearestGridLine(position: number, gridLines: number[], threshold: number): number | null {
  let nearest: number | null = null;
  let minDistance = threshold;

  gridLines.forEach(line => {
    const distance = Math.abs(position - line);
    if (distance < minDistance) {
      minDistance = distance;
      nearest = line;
    }
  });

  return nearest;
}

/** Check if two rectangles overlap */
function checkOverlap(
  rect1: { x: number; y: number; width: number; height: number },
  rect2: { x: number; y: number; width: number; height: number }
): boolean {
  return !(
    rect1.x + rect1.width <= rect2.x ||
    rect2.x + rect2.width <= rect1.x ||
    rect1.y + rect1.height <= rect2.y ||
    rect2.y + rect2.height <= rect1.y
  );
}

/** Find a non-overlapping position near the original position */
function findNonOverlappingPosition(
  terminal: Terminal & { width: number; height: number },
  placedTerminals: (Terminal & { width?: number; height?: number })[],
  positions: Map<number, { x: number; y: number }>,
  minSpacing: number
): { x: number; y: number } {
  const candidates: { x: number; y: number; distance: number }[] = [];
  
  // Try positions around the original position
  const offsets = [
    { dx: 0, dy: terminal.height + minSpacing }, // Below
    { dx: terminal.width + minSpacing, dy: 0 }, // Right
    { dx: 0, dy: -(terminal.height + minSpacing) }, // Above
    { dx: -(terminal.width + minSpacing), dy: 0 }, // Left
  ];

  for (const offset of offsets) {
    const candidateX = terminal.x + offset.dx;
    const candidateY = terminal.y + offset.dy;
    
    const candidateRect = {
      x: candidateX,
      y: candidateY,
      width: terminal.width,
      height: terminal.height
    };

    // Check if this position overlaps with any placed terminal
    const hasOverlap = placedTerminals.some(other => {
      const otherPos = positions.get(other.id);
      if (!otherPos) return false;
      
      return checkOverlap(candidateRect, {
        x: otherPos.x,
        y: otherPos.y,
        width: other.width || 752,
        height: other.height || 515
      });
    });

    if (!hasOverlap) {
      const distance = Math.sqrt(offset.dx * offset.dx + offset.dy * offset.dy);
      candidates.push({ x: candidateX, y: candidateY, distance });
    }
  }

  // Return the closest non-overlapping position, or original if none found
  if (candidates.length > 0) {
    candidates.sort((a, b) => a.distance - b.distance);
    return { x: candidates[0].x, y: candidates[0].y };
  }

  return { x: terminal.x, y: terminal.y };
}

/** Find optimal position for orphaned terminals */
function findOptimalPosition(
  terminal: Terminal & { width: number; height: number },
  placedTerminals: (Terminal & { width?: number; height?: number })[],
  positions: Map<number, { x: number; y: number }>,
  boundingBox: { minX: number; maxX: number; minY: number; maxY: number },
  minSpacing: number
): { x: number; y: number } {
  const candidates: { x: number; y: number; score: number }[] = [];
  
  // If no terminals are placed yet, place at origin
  if (placedTerminals.length === 0) {
    return { x: 0, y: 0 };
  }

  // Try positions in a grid pattern around the bounding box
  const positions_to_try = [
    // Right side of bounding box
    { x: boundingBox.maxX + minSpacing, y: boundingBox.minY },
    // Below bounding box
    { x: boundingBox.minX, y: boundingBox.maxY + minSpacing },
    // Right side, vertically centered
    { x: boundingBox.maxX + minSpacing, y: (boundingBox.minY + boundingBox.maxY - terminal.height) / 2 },
    // Below, horizontally centered
    { x: (boundingBox.minX + boundingBox.maxX - terminal.width) / 2, y: boundingBox.maxY + minSpacing },
  ];

  // Also try to fill gaps in the existing layout
  for (let y = boundingBox.minY; y <= boundingBox.maxY; y += 100) {
    for (let x = boundingBox.minX; x <= boundingBox.maxX; x += 100) {
      positions_to_try.push({ x, y });
    }
  }

  for (const pos of positions_to_try) {
    const candidateRect = {
      x: pos.x,
      y: pos.y,
      width: terminal.width,
      height: terminal.height
    };

    // Check if this position overlaps with any placed terminal
    const hasOverlap = placedTerminals.some(other => {
      const otherPos = positions.get(other.id);
      if (!otherPos) return false;
      
      return checkOverlap(candidateRect, {
        x: otherPos.x,
        y: otherPos.y,
        width: other.width || 752,
        height: other.height || 515
      });
    });

    if (!hasOverlap) {
      // Calculate score based on compactness and alignment
      let score = 0;
      
      // Prefer positions that keep the layout compact
      const centerX = (boundingBox.minX + boundingBox.maxX) / 2;
      const centerY = (boundingBox.minY + boundingBox.maxY) / 2;
      const distToCenter = Math.sqrt(
        Math.pow(pos.x + terminal.width / 2 - centerX, 2) + 
        Math.pow(pos.y + terminal.height / 2 - centerY, 2)
      );
      score -= distToCenter;

      // Prefer positions that align with existing terminals
      placedTerminals.forEach(other => {
        const otherPos = positions.get(other.id);
        if (otherPos) {
          // Bonus for horizontal alignment
          if (Math.abs(pos.y - otherPos.y) < 10) {
            score += 100;
          }
          // Bonus for vertical alignment
          if (Math.abs(pos.x - otherPos.x) < 10) {
            score += 100;
          }
        }
      });

      candidates.push({ x: pos.x, y: pos.y, score });
    }
  }

  // Return the best scoring position
  if (candidates.length > 0) {
    candidates.sort((a, b) => b.score - a.score);
    return { x: candidates[0].x, y: candidates[0].y };
  }

  // Fallback: place to the right of everything
  return { 
    x: boundingBox.maxX + minSpacing, 
    y: boundingBox.minY 
  };
}
