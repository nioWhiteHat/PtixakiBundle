import type { PropertyRequestError } from '../property'

export default function PropertyRequestState({ error, retry, onSignIn, loading }: {
  error?: PropertyRequestError; retry: () => void; onSignIn: () => void; loading: string
}) {
  if (!error) return <div className="property-state" role="status"><span className="property-loading" aria-hidden="true" />{loading}</div>
  return (
    <div className="property-state" role="alert">
      <h2>{error.status === 401 ? 'Sign in to continue' : 'Unable to load this page'}</h2>
      <p>{error.message}</p>
      <button className="property-button" onClick={error.status === 401 ? onSignIn : retry}>
        {error.status === 401 ? 'Sign in again' : 'Try again'}
      </button>
    </div>
  )
}
