"""Numerical validation shared by offline and on-demand native TreeSHAP."""

from __future__ import annotations

import numpy as np


ADDITIVITY_ATOL = 0.001
ADDITIVITY_RTOL = 0.00001


def additivity_tolerance(
    predictions,
    baselines,
    contributions,
    *,
    atol: float = ADDITIVITY_ATOL,
    rtol: float = ADDITIVITY_RTOL,
):
    """Allow float32 rounding on the terms being summed, even near cancellation.

    The native outputs are float32 despite our float64 reconstruction. Scaling
    only by the prediction makes the check arbitrarily strict when large signed
    terms cancel. Use their L1 magnitude (or prediction, whichever is larger).
    This does not adjust the native prediction, baseline, or contributions.
    """
    if not np.isfinite(atol) or not np.isfinite(rtol) or atol < 0 or rtol < 0:
        raise ValueError("SHAP tolerances must be finite and nonnegative")
    contributions = np.asarray(contributions, dtype=np.float64)
    term_magnitude = np.abs(np.asarray(baselines, dtype=np.float64)) + np.abs(
        contributions
    ).sum(axis=-1, dtype=np.float64)
    scale = np.maximum(np.abs(np.asarray(predictions, dtype=np.float64)), term_magnitude)
    return atol + rtol * scale
