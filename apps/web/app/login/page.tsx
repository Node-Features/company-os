import { login } from "./actions";

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const { error } = await searchParams;

  return (
    <main>
      <h1 className="title">CompanyOS</h1>
      <p className="subtitle">Sign in to continue.</p>
      <div className="panel">
        <form action={login} className="login-form">
          <label className="field-label" htmlFor="email">
            Email
          </label>
          <input id="email" name="email" type="email" required className="field-input" />

          <label className="field-label" htmlFor="password">
            Password
          </label>
          <input id="password" name="password" type="password" required className="field-input" />

          <button className="btn btn-accent" type="submit">
            Sign in
          </button>
        </form>
        {error && <p className="error-text">{error}</p>}
      </div>
    </main>
  );
}
