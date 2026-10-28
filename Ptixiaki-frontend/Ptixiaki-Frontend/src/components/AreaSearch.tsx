import { useEffect, useId, useState } from 'react'
import type { KeyboardEvent } from 'react'
import { getPropertyData, PropertyRequestError } from '../property'
import type { AreaSuggestion } from '../analysis'
import './AreaSearch.css'

type AreaSearchProps = {
  selected: AreaSuggestion | null
  onSelect: (area: AreaSuggestion | null) => void
  disabled: boolean
  onSignIn: () => void
  onQueryChange?: (query: string) => void
  kind?: 'area' | 'address'
}

export default function AreaSearch({ selected, onSelect, disabled, onSignIn, onQueryChange, kind = 'area' }: AreaSearchProps) {
  const id = useId()
  const isAddress = kind === 'address'
  const noun = isAddress ? 'address' : 'area'
  const [query, setQuery] = useState(selected?.display_name ?? '')
  const [options, setOptions] = useState<AreaSuggestion[]>([])
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(-1)
  const [status, setStatus] = useState(selected ? `${noun} selected.` : '')
  const [searchVersion, setSearchVersion] = useState(0)
  const [searching, setSearching] = useState(false)
  const [error, setError] = useState<PropertyRequestError | null>(null)

  useEffect(() => {
    if (query.trim().length < 2 || selected || disabled || (isAddress && searchVersion === 0)) {
      return
    }

    const controller = new AbortController()
    const timer = setTimeout(async () => {
      setSearching(true)
      setError(null)
      setStatus(isAddress ? 'Finding addresses…' : 'Finding areas…')

      try {
        const url = `/analysis/autocomplete?q=${encodeURIComponent(query.trim())}${isAddress ? '&kind=address' : ''}`
        const suggestions = await getPropertyData<AreaSuggestion[] | null>(url, controller.signal)

        if (!controller.signal.aborted) {
          setOptions(suggestions ?? [])
          setStatus(suggestions?.length
            ? `Choose an ${noun} from the suggestions.`
            : isAddress
              ? 'No matching addresses found. Include the city or try the Greek street name.'
              : 'No matching areas found. Try its Greek name or a larger municipality.')
        }
      } catch (error) {
        if (!controller.signal.aborted) {
          setError(error as PropertyRequestError)
          setStatus('')
        }
      } finally {
        if (!controller.signal.aborted) {
          setSearching(false)
        }
      }
    }, isAddress ? 0 : 2000)

    return () => {
      clearTimeout(timer)
      controller.abort()
    }
  }, [query, selected, disabled, isAddress, noun, searchVersion])

  function changeQuery(value: string) {
    setQuery(value)
    setSearchVersion(0)
    setSearching(false)
    onQueryChange?.(value)
    onSelect(null)
    setOptions([])
    setActive(-1)
    setError(null)
    setStatus('')
    setOpen(true)
  }

  function choose(area: AreaSuggestion) {
    setQuery(area.display_name)
    onQueryChange?.(area.display_name)
    onSelect(area)
    setOptions([])
    setOpen(false)
    setActive(-1)
    setStatus(`${noun} selected.`)
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Escape') {
      setOpen(false)
      setActive(-1)
    }

    if ((event.key === 'ArrowDown' || event.key === 'ArrowUp') && options.length > 0) {
      event.preventDefault()
      setOpen(true)

      const direction = event.key === 'ArrowDown' ? 1 : -1
      const next = active < 0 ? (direction === 1 ? 0 : options.length - 1) : active + direction
      setActive((next + options.length) % options.length)
    }

    if (event.key === 'Enter' && open && active >= 0 && options[active]) {
      event.preventDefault()
      choose(options[active])
    } else if (event.key === 'Enter' && !selected) {
      event.preventDefault()
      if (isAddress && !searching && query.trim().length >= 2) {
        setSearchVersion((version) => version + 1)
      }
    }
  }

  return (
    <div className="area-search">
      <label htmlFor={id}>{isAddress ? 'Full address' : 'Area name'}</label>
      <input
        id={id}
        role="combobox"
        name={isAddress ? "Address" : undefined}
        required={isAddress}
        maxLength={isAddress ? 255 : undefined}
        autoComplete="off"
        placeholder={isAddress ? 'Search a street address or city' : 'Search an area, e.g. Piraeus'}
        value={query}
        disabled={disabled}
        aria-autocomplete="list"
        aria-expanded={open && options.length > 0}
        aria-controls={`${id}-options`}
        aria-activedescendant={open && active >= 0 ? `${id}-option-${active}` : undefined}
        aria-describedby={`${id}-status`}
        onChange={(event) => changeQuery(event.target.value)}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        onKeyDown={handleKeyDown}
      />
      <ul
        id={`${id}-options`}
        role="listbox"
        aria-label={isAddress ? "Suggested addresses" : "Suggested areas"}
        className="area-suggestions"
        hidden={!open || options.length === 0}
      >
        {options.map((area, index) => (
          <li
            key={`${area.display_name}-${index}`}
            id={`${id}-option-${index}`}
            role="option"
            aria-selected={index === active}
            onMouseDown={(event) => event.preventDefault()}
            onClick={() => choose(area)}
          >
            {area.display_name}
          </li>
        ))}
      </ul>
      {isAddress && (
        <button
          type="button"
          className="property-button address-find-button"
          disabled={disabled || searching || !!selected || query.trim().length < 2}
          onClick={() => {
            setOpen(true)
            setSearchVersion((version) => version + 1)
          }}
        >
          {searching ? 'Finding addresses…' : 'Find address'}
        </button>
      )}
      <p id={`${id}-status`} className="panel-note" role="status">
        {status || (isAddress
          ? 'Enter a street and city, then press Enter or Find address. Choose the matching Nominatim result.'
          : 'Suggestions appear two seconds after you stop typing. Choose a municipality or district.')}
      </p>
      {isAddress && <p className="panel-note">Address search: Nominatim · © OpenStreetMap contributors</p>}
      {error && (
        <div className="area-error" role="alert">
          <p>{error.status === 401 ? error.message : `Couldn’t load ${noun} suggestions. Please try again.`}</p>
          {error.status === 401 && (
            <button type="button" className="property-button" onClick={onSignIn}>
              Sign in again
            </button>
          )}
        </div>
      )}
    </div>
  )
}
